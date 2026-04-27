package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCronTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCronTaskLogic {
	return &DeleteCronTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteCronTaskLogic) DeleteCronTask(req *types.IDReq) (resp *types.BaseResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).DeleteTask(req)
}
