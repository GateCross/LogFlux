package caddy

import (
	"context"
	"time"

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
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := l.svcCtx.DB.Create(server).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
