package caddy

import (
	"context"

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

// GetCaddyConfig 返回当前生效配置；单实例下没有「选哪个节点」这一层。
func (l *GetCaddyConfigLogic) GetCaddyConfig() (resp *types.CaddyConfigResp, err error) {
	config, modules, err := loadCurrentCaddyConfig(l.svcCtx.DB.WithContext(l.ctx))
	if err == nil {
		return &types.CaddyConfigResp{
			Config:  config,
			Modules: modules,
			Hash:    hashConfig(config),
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
