package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyServersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyServersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyServersLogic {
	return &GetCaddyServersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyServersLogic) GetCaddyServers() (resp *types.CaddyServerListResp, err error) {
	var servers []model.CaddyServer
	if err := l.svcCtx.DB.Find(&servers).Error; err != nil {
		return nil, err
	}

	var list []types.CaddyServerItem
	for _, s := range servers {
		list = append(list, types.CaddyServerItem{
			ID:        s.ID,
			Name:      s.Name,
			Url:       s.Url,
			Type:      s.Type,
			CreatedAt: s.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.CaddyServerListResp{
		List: list,
	}, nil
}
