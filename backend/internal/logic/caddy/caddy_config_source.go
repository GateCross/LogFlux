package caddy

import (
	"fmt"
	"strings"

	caddymodel "logflux/model/caddy"

	"gorm.io/gorm"
)

// loadCurrentCaddyConfig 返回 LogFlux 已接管的 Caddy 配置。
// /etc/caddy/Caddyfile 仅作为容器首次启动的 bootstrap 模板，不作为配置来源。
func loadCurrentCaddyConfig(db *gorm.DB) (string, string, error) {
	config, modules := loadCaddyConfigValue(db)
	if strings.TrimSpace(config) == "" {
		return "", emptyModulesJSON, fmt.Errorf("Caddy 配置尚未由 LogFlux 接管，请先保存或导入配置")
	}
	return config, normalizeCaddyModulesJSON(modules), nil
}

// loadCaddyConfigValue 读 system_configs 里的当前配置；未接管时返回空串。
func loadCaddyConfigValue(db *gorm.DB) (string, string) {
	config, modules := caddymodel.LoadConfig(db)
	return config, normalizeCaddyModulesJSON(modules)
}
