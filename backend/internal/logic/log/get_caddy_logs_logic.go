package log

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyLogsLogic {
	return &GetCaddyLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyLogsLogic) GetCaddyLogs(req *types.CaddyLogReq) (resp *types.CaddyLogResp, err error) {
	return service.NewLogService(l.ctx, l.svcCtx).GetCaddyLogs(req)
}
