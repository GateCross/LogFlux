package caddy

import (
	"fmt"
	caddymodel "logflux/model/caddy"
	wafmodel "logflux/model/waf"
	"strings"

	"gorm.io/gorm"
)

func ensureSingleDefaultPolicy(tx *gorm.DB, policy *wafmodel.WafPolicy) error {
	if tx == nil || policy == nil {
		return fmt.Errorf("策略事务无效")
	}
	if !policy.IsDefault {
		return nil
	}

	query := tx.Model(&wafmodel.WafPolicy{}).Where("is_default = ?", true)
	if policy.ID > 0 {
		query = query.Where("id <> ?", policy.ID)
	}
	if err := query.Update("is_default", false).Error; err != nil {
		return fmt.Errorf("重置默认策略失败: %w", err)
	}
	return nil
}

func normalizeCaddyModulesJSON(modules string) string {
	trimmed := strings.TrimSpace(modules)
	if trimmed == "" {
		return emptyModulesJSON
	}
	return trimmed
}

func createCaddyPolicyHistory(tx *gorm.DB, serverID uint, action, config, modules string) error {
	if tx == nil {
		return fmt.Errorf("数据库为空")
	}

	history := &caddymodel.CaddyConfigHistory{
		ServerID: serverID,
		Action:   strings.TrimSpace(action),
		Hash:     hashConfig(config),
		Config:   config,
		Modules:  normalizeCaddyModulesJSON(modules),
	}
	if err := tx.Create(history).Error; err != nil {
		return fmt.Errorf("创建 Caddy 配置历史失败: %w", err)
	}
	return nil
}

func rollbackPolicyConfigToLastGood(server *caddymodel.CaddyServer, lastGoodConfig string) error {
	if server == nil {
		return fmt.Errorf("Caddy 服务器不存在")
	}
	rollbackConfig := strings.TrimSpace(lastGoodConfig)
	if rollbackConfig == "" {
		return fmt.Errorf("last_good 配置为空")
	}

	if err := adaptCaddyfile(server, rollbackConfig); err != nil {
		return fmt.Errorf("回滚 last_good 适配失败: %w", err)
	}
	if err := loadCaddyfile(server, rollbackConfig); err != nil {
		return fmt.Errorf("回滚 last_good 加载失败: %w", err)
	}
	return nil
}
