package notification

import (
	"context"
	"logflux/internal/notification"
	"logflux/internal/svc"
	"logflux/internal/types"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type TestChannelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTestChannelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestChannelLogic {
	return &TestChannelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TestChannelLogic) TestChannel(req *types.TestChannelReq) (resp *types.BaseResp, err error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Test Notification"
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		content = "This is a test notification sent from LogFlux."
	}

	event := notification.NewEvent(
		"system.test",
		notification.LevelInfo,
		title,
		content,
	)
	event.WithData("sent_at", time.Now().Format(time.RFC3339))

	mgr := l.svcCtx.NotificationMgr
	if mgr == nil {
		return &types.BaseResp{Code: 500, Msg: "Notification manager not initialized"}, nil
	}

	sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mgr.SendToChannel(sendCtx, req.ID, event); err != nil {
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "Test notification sent",
	}, nil
}
