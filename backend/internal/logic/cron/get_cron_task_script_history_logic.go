package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronTaskScriptHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCronTaskScriptHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronTaskScriptHistoryLogic {
	return &GetCronTaskScriptHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCronTaskScriptHistoryLogic) GetCronTaskScriptHistory(req *types.CronTaskFileListReq) (resp *types.CronTaskFileListResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).GetScriptHistory(req)
}
