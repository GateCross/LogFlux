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

	if err := l.svcCtx.DB.Create(server).Error; err != nil {
		return nil, err
	}

	// 发送服务器添加通知
	if l.svcCtx.NotificationMgr != nil {
		go l.svcCtx.NotificationMgr.Notify(context.Background(), notification.NewEvent(
			"caddy.server.added",
			notification.LevelInfo,
			"New Caddy Server Added",
			fmt.Sprintf("Server '%s' (%s) was added to the system.", server.Name, server.Url),
		))
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
