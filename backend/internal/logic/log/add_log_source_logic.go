package log

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddLogSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogSourceLogic {
	return &AddLogSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogSourceLogic) AddLogSource(req *types.LogSourceReq) (resp *types.BaseResp, err error) {
	// todo: add your logic here and delete this line

	return
}
