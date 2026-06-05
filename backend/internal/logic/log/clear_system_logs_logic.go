package log

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearSystemLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewClearSystemLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearSystemLogsLogic {
	return &ClearSystemLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ClearSystemLogsLogic) ClearSystemLogs() (resp *types.BaseResp, err error) {
	return service.NewLogService(l.ctx, l.svcCtx).ClearSystemLogs()
}
