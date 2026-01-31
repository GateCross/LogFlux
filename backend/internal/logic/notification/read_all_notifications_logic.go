package notification

import (
	"context"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadAllNotificationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReadAllNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadAllNotificationsLogic {
	return &ReadAllNotificationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReadAllNotificationsLogic) ReadAllNotifications() (resp *types.BaseResp, err error) {
	// Update all unread notifications to read
	err = l.svcCtx.DB.Model(&model.NotificationLog{}).
		Where("is_read = ?", false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		}).Error

	if err != nil {
		l.Logger.Errorf("Failed to mark all notifications as read: %v", err)
		return nil, err
	}

	return &types.BaseResp{
		Code: 0,
		Msg:  "success",
	}, nil
}
