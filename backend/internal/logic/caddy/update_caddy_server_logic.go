package caddy

import (
	"context"
	"fmt"
	"time"

	"logflux/internal/notification"
	"logflux/internal/svc"
	"logflux/internal/types"
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
	if err := l.svcCtx.DB.First(&server, req.ID).Error; err != nil {
		return nil, err
	}

	server.Name = req.Name
	server.Url = req.Url
	server.Token = req.Token
	server.Type = req.Type
	server.Username = req.Username
	server.Password = req.Password
	server.UpdatedAt = time.Now()

	if err := l.svcCtx.DB.Save(&server).Error; err != nil {
		return nil, err
	}

	// 发送配置更新通知
	if l.svcCtx.NotificationMgr != nil {
		go l.svcCtx.NotificationMgr.Notify(context.Background(), notification.NewEvent(
			"caddy.server.updated",
			notification.LevelInfo,
			"Caddy Server Updated",
			fmt.Sprintf("Server '%s' (%s) configuration was updated.", server.Name, server.Url),
		))
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
