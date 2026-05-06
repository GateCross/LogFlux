package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ActivateCronTaskScriptLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewActivateCronTaskScriptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ActivateCronTaskScriptLogic {
	return &ActivateCronTaskScriptLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ActivateCronTaskScriptLogic) ActivateCronTaskScript(req *types.CronTaskFileActivateReq) (resp *types.BaseResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).ActivateTaskScript(req)
}
