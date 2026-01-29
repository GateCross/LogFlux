package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyConfigLogic {
	return &GetCaddyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyConfigLogic) GetCaddyConfig(req *types.CaddyConfigReq) (resp *types.CaddyConfigResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("server not found")
	}

	// Read from Database (Source of Truth)
	if server.Config != "" {
		return &types.CaddyConfigResp{
			Config: server.Config,
		}, nil
	}

	// If DB is empty, return a template or guide
	defaultConfig := `# No Caddyfile found in database.
# Please paste your existing Caddyfile content here.
# It will be saved to the database and pushed to Caddy.
`
	return &types.CaddyConfigResp{
		Config: defaultConfig,
	}, nil

}
