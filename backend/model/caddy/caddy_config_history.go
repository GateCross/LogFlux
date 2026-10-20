package caddy

import "time"

// 配置历史的动作类型。
//
// 取值为 varchar 字段的实际内容，长度上限 32，新增取值不得超出。
const (
	CaddyHistoryActionUpdate              = "update"
	CaddyHistoryActionRollback            = "rollback"
	CaddyHistoryActionSimpleWafApply      = "simple_waf_apply"
	CaddyHistoryActionSimpleWafLastGood   = "simple_waf_last_good"
	CaddyHistoryActionWafIntegrationOn    = "waf_integration_enable"
	CaddyHistoryActionWafIntegrationOff   = "waf_integration_disable"
)

type CaddyConfigHistory struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	ServerID  uint   `gorm:"index;not null"`
	Action    string `gorm:"size:32;default:'update'"`
	Hash      string `gorm:"size:64;index"`
	Config    string `gorm:"type:text"`
	Modules   string `gorm:"type:jsonb"`
}

func (CaddyConfigHistory) TableName() string {
	return "caddy_config_history"
}
