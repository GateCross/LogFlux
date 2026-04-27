package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateCronTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCronTaskLogic {
	return &CreateCronTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateCronTaskLogic) CreateCronTask(req *types.CronTaskReq) (resp *types.BaseResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).CreateTask(req)
}
