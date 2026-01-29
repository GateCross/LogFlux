package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateChannelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateChannelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateChannelLogic {
	return &UpdateChannelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateChannelLogic) UpdateChannel(req *types.ChannelUpdateReq) (resp *types.BaseResp, err error) {
	// todo: add your logic here and delete this line

	return
}
