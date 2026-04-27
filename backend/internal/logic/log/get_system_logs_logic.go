package log

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSystemLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSystemLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSystemLogsLogic {
	return &GetSystemLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSystemLogsLogic) GetSystemLogs(req *types.SystemLogReq) (resp *types.SystemLogResp, err error) {
	return service.NewLogService(l.ctx, l.svcCtx).GetSystemLogs(req)
}
