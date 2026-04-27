package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronLogListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCronLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronLogListLogic {
	return &GetCronLogListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCronLogListLogic) GetCronLogList(req *types.CronLogListReq) (resp *types.CronLogListResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).GetLogList(req)
}
