package cron

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadCronTaskScriptLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadCronTaskScriptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadCronTaskScriptLogic {
	return &UploadCronTaskScriptLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadCronTaskScriptLogic) UploadCronTaskScript(req *types.CronTaskScriptUploadReq) (resp *types.BaseResp, err error) {
	return service.NewCronService(l.ctx, l.svcCtx).UploadTaskScript(req)
}
