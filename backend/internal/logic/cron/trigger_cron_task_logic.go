package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TriggerCronTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTriggerCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TriggerCronTaskLogic {
	return &TriggerCronTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TriggerCronTaskLogic) TriggerCronTask(req *types.TriggerTaskReq) (resp *types.BaseResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).TriggerTask(req)
}
