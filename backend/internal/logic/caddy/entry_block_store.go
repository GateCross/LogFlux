package caddy

import (
	_ "embed"
	"encoding/json"
	"strings"

	"gorm.io/gorm"

	systemmodel "logflux/model/system"
)

// systemConfigKeyCaddyEntryBlock 是 :80 系统入口块模板在 system_configs 里的存储键。
const systemConfigKeyCaddyEntryBlock = "caddy_entry_block"

// defaultCaddyEntryBlock 内置默认 :80 系统入口模板。
//
// 以生产环境打磨过的内网直连入口为蓝本：内网来源按 Host 分流到源站（不跳
// HTTPS，避免 hairpin NAT 回环），其余来源跳 HTTPS。两个坑写在注释里，防止
// 后续编辑时踩回去：route/handle 不可混用会重排，auto_https 需
// disable_redirects 配合（该行由全局选项统一接管）。
//
//go:embed default_entry_block.caddyfile
var defaultCaddyEntryBlock string

// defaultCaddyEntryBlockNormalized 返回去掉首尾空白的默认模板（LF）。
func defaultCaddyEntryBlockNormalized() string {
	return strings.TrimSpace(strings.ReplaceAll(defaultCaddyEntryBlock, "\r\n", "\n"))
}

// caddyEntryBlockValue 入口块模板的存储结构，raw 为 :80 块原文。
type caddyEntryBlockValue struct {
	Raw string `json:"raw"`
}

// loadCaddyEntryBlock 读取入口块模板；键不存在或解析失败时返回内置默认模板。
func loadCaddyEntryBlock(db *gorm.DB) string {
	model := systemmodel.NewSystemConfigModel(db)
	cfg, err := model.GetByKey(systemConfigKeyCaddyEntryBlock)
	if err != nil {
		return defaultCaddyEntryBlockNormalized()
	}
	var value caddyEntryBlockValue
	if err := json.Unmarshal([]byte(cfg.Value), &value); err != nil {
		return defaultCaddyEntryBlockNormalized()
	}
	if strings.TrimSpace(value.Raw) == "" {
		return defaultCaddyEntryBlockNormalized()
	}
	return strings.TrimSpace(strings.ReplaceAll(value.Raw, "\r\n", "\n"))
}

// saveCaddyEntryBlock 保存入口块模板。
func saveCaddyEntryBlock(db *gorm.DB, raw string) error {
	model := systemmodel.NewSystemConfigModel(db)
	encoded, err := json.Marshal(caddyEntryBlockValue{Raw: raw})
	if err != nil {
		return err
	}
	return model.SetByKey(systemConfigKeyCaddyEntryBlock, string(encoded))
}
