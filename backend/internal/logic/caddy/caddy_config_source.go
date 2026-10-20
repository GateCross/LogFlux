package caddy

import (
	"fmt"
	caddymodel "logflux/model/caddy"
)

// loadCurrentCaddyConfig 返回 LogFlux 已接管的 Caddy 配置。
// /etc/caddy/Caddyfile 仅作为容器首次启动的 bootstrap 模板，不作为配置来源。
func loadCurrentCaddyConfig(server *caddymodel.CaddyServer) (string, string, error) {
	if server == nil {
		return "", emptyModulesJSON, fmt.Errorf("Caddy 服务器不存在")
	}

	if config, modules := caddyConfigFromServer(server); config != "" {
		return config, modules, nil
	}

	return "", emptyModulesJSON, fmt.Errorf("Caddy 配置尚未由 LogFlux 接管，请先保存或导入配置")
}
