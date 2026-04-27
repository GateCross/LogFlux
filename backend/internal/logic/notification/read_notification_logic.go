package notification

import (
	"context"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadNotificationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReadNotificationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadNotificationLogic {
	return &ReadNotificationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReadNotificationLogic) ReadNotification(req *types.IDReq) (resp *types.BaseResp, err error) {
	err = l.svcCtx.DB.WithContext(l.ctx).Model(&model.NotificationLog{}).
		Where("id = ?", req.ID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		}).Error

	if err != nil {
		l.Logger.Errorf("读取通知失败: %v", err)
		return nil, err
	}

	return &types.BaseResp{
		Code: 0,
		Msg:  "成功",
	}, nil
}
