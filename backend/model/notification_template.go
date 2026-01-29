package model

import (
	"time"
)

// NotificationTemplate 通知模板模型
type NotificationTemplate struct {
	ID          int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string      `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string      `gorm:"type:varchar(255)" json:"description"`
	Format      string      `gorm:"type:varchar(20);not null" json:"format"` // text, html, markdown, json
	Content     string      `gorm:"type:text;not null" json:"content"`
	Type        string      `gorm:"type:varchar(20);not null;default:'user'" json:"type"` // system, user
	Variables   StringArray `gorm:"type:text" json:"variables"`                           // JSON array of available variables
	CreatedAt   time.Time   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time   `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

// Ensure StringArray is defined in other files (notification_channel.go or notification_rule.go)
// If not, we might need to define it here or verify imports.
// Checking previous progress report, StringArray was implemented in Task 2.
// Assuming it is available in the package `model`.
