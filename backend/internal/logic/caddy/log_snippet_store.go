package caddy

import (
	_ "embed"
	"encoding/json"
	"strings"

	"gorm.io/gorm"

	systemmodel "logflux/model/system"
)

// systemConfigKeyCaddyLogSnippet 是站点日志 snippet 定义在 system_configs 里的存储键。
const systemConfigKeyCaddyLogSnippet = "caddy_log_snippet"

// defaultCaddyLogSnippet 内置默认的日志 snippet 定义。
//
// 参数化写法：name / size / level 三个键由站点侧 import 传值，因此一份定义
// 被所有站点共享；roll_keep 固定 1（0 会被 Caddy 当未设置回落 10）。
//
//go:embed default_log_snippet.caddyfile
var defaultCaddyLogSnippet string

// defaultCaddyLogSnippetNormalized 返回去掉首尾空白的默认定义（LF）。
func defaultCaddyLogSnippetNormalized() string {
	return strings.TrimSpace(strings.ReplaceAll(defaultCaddyLogSnippet, "\r\n", "\n"))
}

// caddyLogSnippetValue 日志 snippet 定义的存储结构，raw 为 snippet 原文。
type caddyLogSnippetValue struct {
	Raw string `json:"raw"`
}

// loadCaddyLogSnippet 读取日志 snippet 定义；键不存在或解析失败时返回内置默认。
func loadCaddyLogSnippet(db *gorm.DB) string {
	model := systemmodel.NewSystemConfigModel(db)
	cfg, err := model.GetByKey(systemConfigKeyCaddyLogSnippet)
	if err != nil {
		return defaultCaddyLogSnippetNormalized()
	}
	var value caddyLogSnippetValue
	if err := json.Unmarshal([]byte(cfg.Value), &value); err != nil {
		return defaultCaddyLogSnippetNormalized()
	}
	if strings.TrimSpace(value.Raw) == "" {
		return defaultCaddyLogSnippetNormalized()
	}
	return strings.TrimSpace(strings.ReplaceAll(value.Raw, "\r\n", "\n"))
}

// saveCaddyLogSnippet 保存日志 snippet 定义。
func saveCaddyLogSnippet(db *gorm.DB, raw string) error {
	model := systemmodel.NewSystemConfigModel(db)
	encoded, err := json.Marshal(caddyLogSnippetValue{Raw: raw})
	if err != nil {
		return err
	}
	return model.SetByKey(systemConfigKeyCaddyLogSnippet, string(encoded))
}
