package model

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// NotificationRule 告警规则模型
type NotificationRule struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 基本信息
	Name        string `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Enabled     bool   `gorm:"default:true;index;not null" json:"enabled"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	// 规则类型
	RuleType string `gorm:"size:50;not null" json:"rule_type"` // threshold, frequency, ratio, pattern, composite

	// 条件表达式 (JSONB)
	Condition JSONMap `gorm:"type:jsonb;not null" json:"condition"`

	// 触发事件类型
	EventType string `gorm:"size:100;index;not null" json:"event_type"`

	// 通知渠道 ID
	ChannelIDs Int64Array `gorm:"type:bigint[];not null;default:'{}'" json:"channel_ids"`

	// 通知模板 (可选)
	Template string `gorm:"type:text" json:"template,omitempty"`

	// 静默时间 (秒)
	SilenceDuration int `gorm:"default:300" json:"silence_duration"`

	// 最后触发时间
	LastTriggeredAt *time.Time `gorm:"index" json:"last_triggered_at,omitempty"`

	// 触发次数
	TriggerCount int `gorm:"default:0" json:"trigger_count"`
}

// TableName 返回表名
func (NotificationRule) TableName() string {
	return "notification_rules"
}

// Int64Array 自定义类型,用于 BIGINT[] 字段
type Int64Array []int64

// Value 实现 driver.Valuer 接口
func (i Int64Array) Value() (driver.Value, error) {
	if i == nil {
		return "{}", nil
	}
	// PostgreSQL array format: {1,2,3}
	result := "{"
	for idx, v := range i {
		if idx > 0 {
			result += ","
		}
		result += fmt.Sprintf("%d", v)
	}
	result += "}"
	return result, nil
}

// Scan 实现 sql.Scanner 接口
func (i *Int64Array) Scan(value interface{}) error {
	if value == nil {
		*i = []int64{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("failed to scan Int64Array value, type: %T", value)
	}

	if str == "{}" || str == "" {
		*i = []int64{}
		return nil
	}

	// Remove braces and split
	str = str[1 : len(str)-1]
	parts := strings.Split(str, ",")
	result := make([]int64, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil {
			return err
		}
		result = append(result, v)
	}
	*i = result
	return nil
}

// RuleType 规则类型常量
const (
	RuleTypeThreshold = "threshold" // 阈值规则
	RuleTypeFrequency = "frequency" // 频率规则
	RuleTypeRatio     = "ratio"     // 比率规则
	RuleTypePattern   = "pattern"   // 模式匹配规则
	RuleTypeComposite = "composite" // 复合规则
)

// ThresholdCondition 阈值规则条件
type ThresholdCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // >, <, >=, <=, ==, !=
	Value    interface{} `json:"value"`
}

// FrequencyCondition 频率规则条件
type FrequencyCondition struct {
	Event   string `json:"event"`              // 事件类型
	Count   int    `json:"count"`              // 次数阈值
	Window  string `json:"window"`             // 时间窗口,如 "5m", "1h"
	GroupBy string `json:"group_by,omitempty"` // 分组字段,如 "remote_ip"
}

// RatioCondition 比率规则条件
type RatioCondition struct {
	Numerator   string  `json:"numerator"`   // 分子条件表达式
	Denominator string  `json:"denominator"` // 分母条件表达式
	Threshold   float64 `json:"threshold"`   // 阈值 (0.0 - 1.0)
	Window      string  `json:"window"`      // 时间窗口
}

// PatternCondition 模式匹配规则条件
type PatternCondition struct {
	Field   string `json:"field"`   // 字段名
	Pattern string `json:"pattern"` // 正则表达式
}

// CompositeCondition 复合规则条件
type CompositeCondition struct {
	Operator   string        `json:"operator"`   // AND, OR
	Conditions []interface{} `json:"conditions"` // 子条件列表
}
