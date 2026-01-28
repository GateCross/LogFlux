package model

import (
	"time"
)

// NotificationLog 通知历史记录模型
type NotificationLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `gorm:"index:idx_created_at,sort:desc" json:"created_at"`

	// 关联的渠道和规则
	ChannelID uint  `gorm:"index;not null" json:"channel_id"`
	RuleID    *uint `gorm:"index" json:"rule_id,omitempty"` // 可选,手动发送的通知没有规则

	// 事件信息
	EventType string  `gorm:"size:100;index;not null" json:"event_type"`
	EventData JSONMap `gorm:"type:jsonb" json:"event_data,omitempty"`

	// 发送状态
	Status       string     `gorm:"size:50;index;not null;default:'pending'" json:"status"` // pending, success, failed
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
	SentAt       *time.Time `json:"sent_at,omitempty"`

	// 关联
	Channel *NotificationChannel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	Rule    *NotificationRule    `gorm:"foreignKey:RuleID" json:"rule,omitempty"`
}

// TableName 返回表名
func (NotificationLog) TableName() string {
	return "notification_logs"
}

// NotificationStatus 通知状态常量
const (
	NotificationStatusPending = "pending"
	NotificationStatusSuccess = "success"
	NotificationStatusFailed  = "failed"
)
