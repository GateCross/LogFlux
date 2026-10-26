package system

import (
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

// 日志保留策略在 system_configs 里的存储键。
const LogRetentionConfigKey = "caddy_log_retention"

// 保留天数缺省值与允许上限；UI 校验与清理任务共用。
const (
	DefaultLogRetentionDays = 7
	MaxLogRetentionDays     = 3650
)

// LogRetention 站点日志保留策略；Days 取值 1..MaxLogRetentionDays。
type LogRetention struct {
	Enabled bool
	Days    int
}

// 默认策略：开启保留并保留 7 天。
func defaultLogRetention() LogRetention {
	return LogRetention{Enabled: true, Days: DefaultLogRetentionDays}
}

// LoadLogRetention 宽容读取保留策略；键不存在、JSON 解析失败或 Days 越界时返回默认策略。
// 供查询接口使用，任何情况下都有可展示的值。
func LoadLogRetention(db *gorm.DB) LogRetention {
	retention, err := loadLogRetention(db)
	if err != nil {
		return defaultLogRetention()
	}
	return retention
}

// LoadLogRetentionStrict 供清理任务等破坏性路径使用：读库失败不回落默认，
// 原样返回 error，由调用方跳过本轮并告警。只有键不存在才按默认策略处理。
func LoadLogRetentionStrict(db *gorm.DB) (LogRetention, error) {
	return loadLogRetention(db)
}

// loadLogRetention 读库并解析保留策略；除键不存在外，读库错误原样返回。
func loadLogRetention(db *gorm.DB) (LogRetention, error) {
	var cfg SystemConfig
	err := db.Where("key = ?", LogRetentionConfigKey).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultLogRetention(), nil
		}
		return LogRetention{}, err
	}
	return parseLogRetention(cfg.Value)
}

// parseLogRetention 解析存储值；内容非法或 Days 越界一律按默认策略处理。
func parseLogRetention(raw string) (LogRetention, error) {
	var value struct {
		Enabled bool `json:"enabled"`
		Days    int  `json:"days"`
	}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return defaultLogRetention(), nil
	}
	if value.Days < 1 || value.Days > MaxLogRetentionDays {
		return defaultLogRetention(), nil
	}
	return LogRetention{Enabled: value.Enabled, Days: value.Days}, nil
}

// SaveLogRetention 保存保留策略。
func SaveLogRetention(db *gorm.DB, r LogRetention) error {
	encoded, err := json.Marshal(struct {
		Enabled bool `json:"enabled"`
		Days    int  `json:"days"`
	}{Enabled: r.Enabled, Days: r.Days})
	if err != nil {
		return err
	}
	return NewSystemConfigModel(db).SetByKey(LogRetentionConfigKey, string(encoded))
}
