package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronTaskListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCronTaskListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronTaskListLogic {
	return &GetCronTaskListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCronTaskListLogic) GetCronTaskList(req *types.CronTaskListReq) (resp *types.CronTaskListResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).GetTaskList(req)
}
