package caddy

import (
	"context"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetWafEngineStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetWafEngineStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWafEngineStatusLogic {
	return &GetWafEngineStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetWafEngineStatusLogic) GetWafEngineStatus() (resp *types.WafEngineStatusResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)
	l.svcCtx.EnsureWafEngineDefaultSource()

	var source model.WafSource
	if err = helper.svcCtx.DB.Where("kind = ?", wafKindCorazaEngine).Order("updated_at desc, id desc").First(&source).Error; err != nil {
		return &types.WafEngineStatusResp{
			CurrentVersion: "",
			LatestVersion:  "",
			CanUpgrade:     false,
			CheckedAt:      "",
			Source:         "",
			Message:        "未找到 Coraza 引擎更新源，请先在更新源配置中新增 coraza_engine 类型源",
		}, nil
	}

	currentVersion := strings.TrimSpace(source.LastRelease)
	latestVersion := strings.TrimSpace(source.LastRelease)

	var activeRelease model.WafRelease
	if activeErr := helper.svcCtx.DB.
		Where("kind = ? AND status = ?", wafKindCorazaEngine, wafReleaseStatusActive).
		Order("updated_at desc, id desc").
		First(&activeRelease).Error; activeErr == nil {
		if version := strings.TrimSpace(activeRelease.Version); version != "" {
			currentVersion = version
		}
	}

	var latestRelease model.WafRelease
	if latestErr := helper.svcCtx.DB.
		Where("kind = ?", wafKindCorazaEngine).
		Order("created_at desc, id desc").
		First(&latestRelease).Error; latestErr == nil {
		if version := strings.TrimSpace(latestRelease.Version); version != "" {
			latestVersion = version
		}
	}

	if latestVersion == "" {
		latestVersion = currentVersion
	}

	canUpgrade := false
	if currentVersion != "" && latestVersion != "" {
		canUpgrade = currentVersion != latestVersion
	}

	checkedAt := formatNullableTime(source.LastCheckedAt)
	if checkedAt == "" {
		checkedAt = formatTime(source.UpdatedAt)
	}

	message := strings.TrimSpace(source.LastError)
	if message == "" {
		if currentVersion == "" && latestVersion == "" {
			message = "暂未发现 Coraza 引擎版本记录，请先执行检查或同步"
		} else {
			message = "引擎版本状态已读取"
		}
	}

	return &types.WafEngineStatusResp{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		CanUpgrade:     canUpgrade,
		CheckedAt:      checkedAt,
		Source:         source.Name,
		Message:        message,
	}, nil

}
