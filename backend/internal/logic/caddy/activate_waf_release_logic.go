package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ActivateWafReleaseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewActivateWafReleaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ActivateWafReleaseLogic {
	return &ActivateWafReleaseLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ActivateWafReleaseLogic) ActivateWafRelease(req *types.WafReleaseActivateReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	var release model.WafRelease
	if err := helper.svcCtx.DB.WithContext(helper.ctx).First(&release, req.ID).Error; err != nil {
		return nil, fmt.Errorf("版本不存在")
	}
	if normalizeWafKind(release.Kind) == wafKindCorazaEngine {
		return nil, fmt.Errorf("Coraza 引擎不支持在线激活，仅支持版本检查")
	}

	job := helper.startJob(release.SourceID, release.ID, "activate", "manual")

	if err := helper.activateRelease(&release); err != nil {
		helper.markReleaseFailed(&release, err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), release.ID)
		return nil, err
	}

	if err := helper.markReleaseActive(&release); err != nil {
		helper.finishJob(job, wafJobStatusFailed, err.Error(), release.ID)
		return nil, fmt.Errorf("标记激活状态失败: %w", err)
	}

	helper.finishJob(job, wafJobStatusSuccess, "激活成功", release.ID)
	return &types.BaseResp{Code: 200, Msg: "成功"}, nil
}
