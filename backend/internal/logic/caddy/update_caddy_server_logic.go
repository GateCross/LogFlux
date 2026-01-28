package caddy

import (
	"context"
	"time"

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

func (l *UpdateCaddyServerLogic) UpdateCaddyServer(req *types.CaddyServerReq) (resp *types.BaseResp, err error) {
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

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
