package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteNotificationLogLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteNotificationLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteNotificationLogLogic {
	return &DeleteNotificationLogLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteNotificationLogLogic) DeleteNotificationLog(req *types.IDReq) (resp *types.BaseResp, err error) {
	// delete related job first
	l.svcCtx.DB.Where("log_id = ?", req.ID).Delete(&model.NotificationJob{})

	if err := l.svcCtx.DB.Delete(&model.NotificationLog{}, req.ID).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
