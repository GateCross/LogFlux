package caddy

import (
	"context"
	"fmt"
	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"

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
	var server caddymodel.CaddyServer
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("服务器不存在")
	}

	config, modules, err := loadCurrentCaddyConfig(&server)
	if err == nil {
		return &types.CaddyConfigResp{
			Config:  config,
			Modules: modules,
		}, nil
	}

	defaultConfig := `# LogFlux 尚未接管 Caddy 配置。
# /etc/caddy/Caddyfile 仅作为首次启动模板，Caddy autosave 仅作为运行态缓存。
# 请粘贴当前 Caddyfile 后保存；保存成功后数据库将作为管理态唯一来源。
`

	return &types.CaddyConfigResp{
		Config:  defaultConfig,
		Modules: emptyModulesJSON,
	}, nil

}
