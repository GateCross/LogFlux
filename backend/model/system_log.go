package model

import "time"

// SystemLog 用于存储后端与 Caddy 后台运行日志
type SystemLog struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time

	LogTime time.Time `gorm:"index:idx_system_log_time_source,priority:1;index:idx_system_log_time;not null"`
	Level   string    `gorm:"size:20;index:idx_system_log_level"`
	Message string    `gorm:"type:text"`
	Caller  string    `gorm:"size:255"`
	TraceID string    `gorm:"size:128;index"`
	SpanID  string    `gorm:"size:128"`

	Source   string `gorm:"size:50;index:idx_system_log_time_source,priority:2"` // backend | caddy_runtime
	FilePath string `gorm:"size:1024"`

	RawLog    string `gorm:"type:text;comment:原始完整日志"`
	ExtraData string `gorm:"type:jsonb;comment:扩展元数据"`
}

func (SystemLog) TableName() string {
	return "system_logs"
}
