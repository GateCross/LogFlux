package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckWafEngineLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckWafEngineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckWafEngineLogic {
	return &CheckWafEngineLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckWafEngineLogic) CheckWafEngine() (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)
	l.svcCtx.EnsureWafEngineDefaultSource()

	var source model.WafSource
	if err = helper.svcCtx.DB.Where("kind = ?", wafKindCorazaEngine).Order("updated_at desc, id desc").First(&source).Error; err != nil {
		return nil, fmt.Errorf("engine source not found")
	}

	job := helper.startJob(source.ID, 0, "engine_check", "manual")
	if strings.TrimSpace(source.URL) == "" {
		err = fmt.Errorf("engine source url is empty")
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	releaseVersion := strings.TrimSpace(source.LastRelease)
	var latestRelease model.WafRelease
	if queryErr := helper.svcCtx.DB.Where("kind = ?", wafKindCorazaEngine).Order("created_at desc, id desc").First(&latestRelease).Error; queryErr == nil {
		if version := strings.TrimSpace(latestRelease.Version); version != "" {
			releaseVersion = version
		}
	}

	// 当前仅做可用性占位检查；真实版本解析后续接入
	helper.updateSourceLastCheck(source.ID, releaseVersion, "")
	helper.finishJob(job, wafJobStatusSuccess, "engine source check success", 0)

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
