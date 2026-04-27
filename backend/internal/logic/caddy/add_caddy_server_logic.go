package caddy

import (
	"context"
	"fmt"
	"time"

	"logflux/internal/notification"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/safego"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddCaddyServerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddCaddyServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddCaddyServerLogic {
	return &AddCaddyServerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddCaddyServerLogic) AddCaddyServer(req *types.CaddyServerReq) (resp *types.BaseResp, err error) {
	server := &model.CaddyServer{
		Name:      req.Name,
		Url:       req.Url,
		Token:     req.Token,
		Type:      req.Type,
		Username:  req.Username,
		Password:  req.Password,
		Modules:   "{}",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := l.svcCtx.DB.WithContext(l.ctx).Create(server).Error; err != nil {
		return nil, err
	}

	// 发送服务器添加通知
	if l.svcCtx.NotificationMgr != nil {
		event := notification.NewEvent(
			"caddy.server.added",
			notification.LevelInfo,
			"新增 Caddy 服务器",
			fmt.Sprintf("Caddy 服务器 %s（%s）已添加。", server.Name, server.Url),
		)
		safego.New(context.Background(), "新增 Caddy 服务器通知").Go(func() {
			if err := l.svcCtx.NotificationMgr.Notify(context.Background(), event); err != nil {
				l.Errorf("发送 Caddy 服务器新增通知失败: serverID=%d err=%v", server.ID, err)
			}
		})
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
