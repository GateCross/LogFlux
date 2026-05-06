package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronLogDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCronLogDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronLogDetailLogic {
	return &GetCronLogDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCronLogDetailLogic) GetCronLogDetail(req *types.CronLogDetailReq) (resp *types.CronLogItem, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).GetLogDetail(req)
}
