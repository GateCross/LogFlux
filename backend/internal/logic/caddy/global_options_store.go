package caddy

import (
	"encoding/json"

	"gorm.io/gorm"

	systemmodel "logflux/model/system"
)

// systemConfigKeyCaddyGlobalOptions 是全局选项在 system_configs 里的存储键。
const systemConfigKeyCaddyGlobalOptions = "caddy_global_options"

// CaddyGlobalOptionsValue 全局选项存储结构（单独保存，写入时合并进全局块）。
//
// 接管语义：仅显式配置了值的键参与合并（同名指令删后写入）；未配置的键
// 不输出、也不动配置里既有内容。Debug 用指针区分"未设置"与"false"。
type CaddyGlobalOptionsValue struct {
	AutoHttps       string   `json:"autoHttps,omitempty"`
	TrustedProxies  []string `json:"trustedProxies,omitempty"`
	Debug           *bool    `json:"debug,omitempty"`
	Email           string   `json:"email,omitempty"`
	ExtraRawOptions string   `json:"extraRawOptions,omitempty"`
}

// defaultCaddyGlobalOptions 首次读取时的缺省值（不写入，等用户显式保存）。
func defaultCaddyGlobalOptions() CaddyGlobalOptionsValue {
	return CaddyGlobalOptionsValue{}
}

// loadCaddyGlobalOptions 读取全局选项；键不存在时返回缺省值。
func loadCaddyGlobalOptions(db *gorm.DB) CaddyGlobalOptionsValue {
	model := systemmodel.NewSystemConfigModel(db)
	cfg, err := model.GetByKey(systemConfigKeyCaddyGlobalOptions)
	if err != nil {
		return defaultCaddyGlobalOptions()
	}
	var value CaddyGlobalOptionsValue
	if err := json.Unmarshal([]byte(cfg.Value), &value); err != nil {
		return defaultCaddyGlobalOptions()
	}
	return value
}

// saveCaddyGlobalOptions 保存全局选项。
func saveCaddyGlobalOptions(db *gorm.DB, value CaddyGlobalOptionsValue) error {
	model := systemmodel.NewSystemConfigModel(db)
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return model.SetByKey(systemConfigKeyCaddyGlobalOptions, string(encoded))
}
