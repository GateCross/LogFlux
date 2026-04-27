package log

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogSourcesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogSourcesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogSourcesLogic {
	return &ListLogSourcesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogSourcesLogic) ListLogSources(req *types.LogSourceListReq) (resp *types.LogSourceListResp, err error) {
	return service.NewLogSourceService(l.ctx, l.svcCtx).List(req)
}
