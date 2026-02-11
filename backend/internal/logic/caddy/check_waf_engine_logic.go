package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"

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
	job := helper.startJob(0, 0, "engine_check", "manual")
	latestVersion, fetchErr := helper.fetchCorazaLatestReleaseVersion()
	if fetchErr != nil {
		helper.finishJob(job, wafJobStatusFailed, fetchErr.Error(), 0)
		return nil, fetchErr
	}

	if job != nil {
		if err := helper.svcCtx.DB.Model(job).Updates(map[string]interface{}{
			"meta":    map[string]interface{}{"latestVersion": latestVersion},
			"message": fmt.Sprintf("engine source check success: latest=%s", latestVersion),
		}).Error; err != nil {
			helper.logger.Errorf("update engine check job meta failed: %v", err)
		}
	}
	helper.finishJob(job, wafJobStatusSuccess, fmt.Sprintf("engine source check success: latest=%s", latestVersion), 0)

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
