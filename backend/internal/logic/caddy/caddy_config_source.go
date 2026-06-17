package caddy

import (
	"fmt"
	caddymodel "logflux/model/caddy"
	"os"
	"strings"
)

// localCaddyfilePath 仅预留给后续显式导入 bootstrap Caddyfile 使用。
// 当前配置读取不会自动 fallback 到该文件。
var localCaddyfilePath = "/etc/caddy/Caddyfile"

// loadCurrentCaddyConfig 返回 LogFlux 已接管的 Caddy 配置。
// /etc/caddy/Caddyfile 仅作为容器首次启动的 bootstrap 模板，不再作为当前配置来源。
func loadCurrentCaddyConfig(server *caddymodel.CaddyServer) (string, string, error) {
	if server == nil {
		return "", emptyModulesJSON, fmt.Errorf("Caddy 服务器不存在")
	}

	if config, modules := caddyConfigFromServer(server); config != "" {
		return config, modules, nil
	}

	return "", emptyModulesJSON, fmt.Errorf("Caddy 配置尚未由 LogFlux 接管，请先保存或导入配置")
}

// readLocalCaddyfile 读取本地 Caddyfile，返回值会自动去掉首尾空白。
func readLocalCaddyfile() (string, error) {
	raw, err := os.ReadFile(localCaddyfilePath)
	if err != nil {
		return "", err
	}

	config := strings.TrimSpace(string(raw))
	if config == "" {
		return "", fmt.Errorf("本地 Caddyfile 为空")
	}
	return config, nil
}
