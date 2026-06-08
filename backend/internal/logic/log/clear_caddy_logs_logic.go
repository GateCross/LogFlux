package log

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearCaddyLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewClearCaddyLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearCaddyLogsLogic {
	return &ClearCaddyLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ClearCaddyLogsLogic) ClearCaddyLogs() (resp *types.BaseResp, err error) {
	return service.NewLogService(l.ctx, l.svcCtx).ClearCaddyLogs()
}
