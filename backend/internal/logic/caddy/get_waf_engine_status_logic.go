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
	currentVersion := helper.corazaCurrentVersion()

	latestVersion := ""
	var latestCheckJob model.WafUpdateJob
	if queryErr := helper.svcCtx.DB.
		Where("action = ? AND status = ?", "engine_check", wafJobStatusSuccess).
		Order("finished_at desc, id desc").
		First(&latestCheckJob).Error; queryErr == nil {
		latestVersion = latestEngineCheckVersion(&latestCheckJob)
	}

	canUpgrade := false
	if currentVersion != "" && latestVersion != "" {
		canUpgrade = currentVersion != latestVersion
	}

	checkedAt := formatNullableTime(latestCheckJob.FinishedAt)
	if checkedAt == "" {
		checkedAt = formatTime(latestCheckJob.UpdatedAt)
	}

	message := strings.TrimSpace(latestCheckJob.Message)
	if message == "" {
		if currentVersion == "" && latestVersion == "" {
			message = "暂未获取到 Coraza 引擎版本，请点击“检查上游版本”"
		} else if latestVersion == "" {
			message = "已读取当前版本，请点击“检查上游版本”获取最新 Release"
		} else {
			message = "引擎版本状态已读取"
		}
	}

	source := helper.corazaReleaseAPI()
	if source == "" {
		source = defaultCorazaReleaseAPI
	}

	return &types.WafEngineStatusResp{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		CanUpgrade:     canUpgrade,
		CheckedAt:      checkedAt,
		Source:         source,
		Message:        message,
	}, nil

}
