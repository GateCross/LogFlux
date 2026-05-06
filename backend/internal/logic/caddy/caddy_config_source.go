package caddy

import (
	"fmt"
	caddymodel "logflux/model/caddy"
	"os"
	"strings"
)

// 默认读取本地 Caddyfile 的路径，测试可以临时覆盖。
var localCaddyfilePath = "/etc/caddy/Caddyfile"

// loadCurrentCaddyConfig 统一返回当前可用的 Caddy 配置。
// 只有数据库里已经确认的结构化快照才保留 modules；本地文件回退一律返回空快照。
func loadCurrentCaddyConfig(server *caddymodel.CaddyServer) (string, string, error) {
	if server == nil {
		return "", emptyModulesJSON, fmt.Errorf("Caddy 服务器不存在")
	}

	if config, modules := caddyConfigFromServer(server); config != "" {
		return config, modules, nil
	}

	if strings.EqualFold(server.Type, "local") {
		config, err := readLocalCaddyfile()
		if err == nil {
			return config, emptyModulesJSON, nil
		}
	}

	return "", emptyModulesJSON, fmt.Errorf("Caddy 配置为空，请先保存 Caddy 配置")
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
