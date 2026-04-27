package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCronTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCronTaskLogic {
	return &UpdateCronTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCronTaskLogic) UpdateCronTask(req *types.CronTaskUpdateReq) (resp *types.BaseResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).UpdateTask(req)
}
