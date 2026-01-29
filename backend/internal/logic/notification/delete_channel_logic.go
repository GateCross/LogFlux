package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteChannelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteChannelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteChannelLogic {
	return &DeleteChannelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteChannelLogic) DeleteChannel(req *types.IDReq) (resp *types.BaseResp, err error) {
	if err := l.svcCtx.DB.Delete(&model.NotificationChannel{}, req.ID).Error; err != nil {
		return nil, err
	}

	// Reload channels
	if l.svcCtx.NotificationMgr != nil {
		l.svcCtx.NotificationMgr.ReloadChannels()
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
