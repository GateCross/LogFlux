package log

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteLogSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteLogSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteLogSourceLogic {
	return &DeleteLogSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteLogSourceLogic) DeleteLogSource(req *types.IDReq) (resp *types.BaseResp, err error) {
	// todo: add your logic here and delete this line

	return
}
