package log

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLogSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogSourceLogic {
	return &UpdateLogSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogSourceLogic) UpdateLogSource(req *types.LogSourceUpdateReq) (resp *types.BaseResp, err error) {
	return service.NewLogSourceService(l.ctx, l.svcCtx).Update(req)
}
