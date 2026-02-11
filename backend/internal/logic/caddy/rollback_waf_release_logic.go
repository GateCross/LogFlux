package caddy

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RollbackWafReleaseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRollbackWafReleaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RollbackWafReleaseLogic {
	return &RollbackWafReleaseLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RollbackWafReleaseLogic) RollbackWafRelease(req *types.WafReleaseRollbackReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	if err := helper.ensureStoreDirs(); err != nil {
		return nil, err
	}

	targetRelease, err := l.resolveRollbackTarget(helper, req)
	if err != nil {
		return nil, err
	}
	if normalizeWafKind(targetRelease.Kind) == wafKindCorazaEngine {
		return nil, fmt.Errorf("Coraza 引擎不支持在线激活，仅支持版本检查")
	}

	job := helper.startJob(targetRelease.SourceID, targetRelease.ID, "rollback", "manual")

	if err := helper.activateRelease(targetRelease); err != nil {
		helper.markReleaseFailed(targetRelease, err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), targetRelease.ID)
		return nil, err
	}

	if err := helper.markReleaseActive(targetRelease); err != nil {
		helper.finishJob(job, wafJobStatusFailed, err.Error(), targetRelease.ID)
		return nil, fmt.Errorf("mark rollback active failed: %w", err)
	}

	helper.finishJob(job, wafJobStatusSuccess, "rollback success", targetRelease.ID)
	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}

func (l *RollbackWafReleaseLogic) resolveRollbackTarget(helper *wafLogicHelper, req *types.WafReleaseRollbackReq) (*model.WafRelease, error) {
	if version := strings.TrimSpace(req.Version); version != "" {
		return l.findReleaseByVersion(helper, version)
	}

	if strings.EqualFold(strings.TrimSpace(req.Target), "version") {
		return nil, fmt.Errorf("version is required when target=version")
	}

	lastGoodPath, err := helper.store.LinkTarget(helper.store.LastGoodLinkPath())
	if err != nil {
		return nil, fmt.Errorf("last_good link not found")
	}

	version := filepath.Base(lastGoodPath)
	if version == "" || version == "." || version == "/" {
		return nil, fmt.Errorf("invalid last_good target")
	}
	return l.findReleaseByVersion(helper, version)
}

func (l *RollbackWafReleaseLogic) findReleaseByVersion(helper *wafLogicHelper, version string) (*model.WafRelease, error) {
	var release model.WafRelease
	err := helper.svcCtx.DB.Where("version = ?", strings.TrimSpace(version)).Order("id desc").First(&release).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("target release not found")
		}
		return nil, fmt.Errorf("query target release failed: %w", err)
	}
	return &release, nil
}
