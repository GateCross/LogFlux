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

type UpdateCaddyServerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyServerLogic {
	return &UpdateCaddyServerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCaddyServerLogic) UpdateCaddyServer(req *types.UpdateCaddyServerReq) (resp *types.BaseResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&server, req.ID).Error; err != nil {
		return nil, err
	}

	server.Name = req.Name
	server.Url = req.Url
	server.Token = req.Token
	server.Type = req.Type
	server.Username = req.Username
	server.Password = req.Password
	server.UpdatedAt = time.Now()

	if err := l.svcCtx.DB.WithContext(l.ctx).Save(&server).Error; err != nil {
		return nil, err
	}

	// 发送配置更新通知
	if l.svcCtx.NotificationMgr != nil {
		event := notification.NewEvent(
			"caddy.server.updated",
			notification.LevelInfo,
			"更新 Caddy 服务器",
			fmt.Sprintf("Caddy 服务器 %s（%s）配置已更新。", server.Name, server.Url),
		)
		safego.New(context.Background(), "更新 Caddy 服务器通知").Go(func() {
			if err := l.svcCtx.NotificationMgr.Notify(context.Background(), event); err != nil {
				l.Errorf("发送 Caddy 服务器更新通知失败: serverID=%d err=%v", server.ID, err)
			}
		})
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
