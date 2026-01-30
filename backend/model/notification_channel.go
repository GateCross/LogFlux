package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// NotificationChannel 通知渠道模型
type NotificationChannel struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 基本信息
	Name        string `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Type        string `gorm:"size:50;index;not null" json:"type"` // webhook, email, telegram, slack, wecom, dingtalk
	Enabled     bool   `gorm:"default:true;index;not null" json:"enabled"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	// 配置 (JSONB)
	Config JSONMap `gorm:"type:jsonb;not null" json:"config"`

	// 事件过滤 (TEXT[])
	Events StringArray `gorm:"type:text[];not null;default:'{}'" json:"events"`
}

// TableName 返回表名
func (NotificationChannel) TableName() string {
	return "notification_channels"
}

// JSONMap 自定义类型,用于 JSONB 字段
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}

	return json.Unmarshal(bytes, j)
}

// StringArray 自定义类型,用于 TEXT[] 字段
type StringArray []string

// Value 实现 driver.Valuer 接口
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return "{}", nil
	}
	// PostgreSQL array format: {"value1","value2"}
	return "{" + joinStringArray(s, ",") + "}", nil
}

// Scan 实现 sql.Scanner 接口
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("failed to scan StringArray value")
	}

	// Parse PostgreSQL array format
	str := string(bytes)
	if str == "{}" {
		*s = []string{}
		return nil
	}

	// Remove braces and split
	str = str[1 : len(str)-1]
	*s = splitStringArray(str)
	return nil
}

// Helper functions
func joinStringArray(arr []string, sep string) string {
	if len(arr) == 0 {
		return ""
	}
	result := ""
	for i, s := range arr {
		if i > 0 {
			result += sep
		}
		result += "\"" + s + "\""
	}
	return result
}

func splitStringArray(str string) []string {
	if str == "" {
		return []string{}
	}
	// Simple split by comma (doesn't handle escaped commas)
	result := []string{}
	current := ""
	inQuote := false

	for _, c := range str {
		if c == '"' {
			inQuote = !inQuote
		} else if c == ',' && !inQuote {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// BeforeSave GORM hook to ensure Config is valid JSON before saving
func (c *NotificationChannel) BeforeSave(tx *gorm.DB) error {
	if c.Config == nil {
		c.Config = make(JSONMap)
	}
	return nil
}

// ChannelType 渠道类型常量
const (
	ChannelTypeWebhook  = "webhook"
	ChannelTypeEmail    = "email"
	ChannelTypeTelegram = "telegram"
	ChannelTypeSlack    = "slack"
	ChannelTypeWeCom    = "wecom"
	ChannelTypeDingTalk = "dingtalk"
	ChannelTypeDiscord  = "discord"
)

// Webhook 配置结构
type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"` // POST, GET, PUT
	Headers map[string]string `json:"headers,omitempty"`
}

// Email 配置结构
type EmailConfig struct {
	SmtpHost string   `json:"smtp_host"`
	SmtpPort int      `json:"smtp_port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
}

// Telegram 配置结构
type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

// Slack 配置结构
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// WeCom 配置结构
type WeComConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// DingTalk 配置结构
type DingTalkConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret,omitempty"`
}

// Discord 配置结构
type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`
	Username   string `json:"username,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
}
