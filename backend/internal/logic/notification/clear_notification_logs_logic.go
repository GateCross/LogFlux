package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearNotificationLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewClearNotificationLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearNotificationLogsLogic {
	return &ClearNotificationLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ClearNotificationLogsLogic) ClearNotificationLogs() (resp *types.BaseResp, err error) {
	// clear jobs first
	if err := l.svcCtx.DB.Exec("TRUNCATE TABLE notification_jobs RESTART IDENTITY").Error; err != nil {
		return nil, err
	}
	if err := l.svcCtx.DB.Exec("TRUNCATE TABLE notification_logs RESTART IDENTITY").Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}

var _ = model.NotificationLog{}
