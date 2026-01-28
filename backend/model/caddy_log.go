package model

import (
	"time"
)

type CaddyLog struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"` // Index for time-range queries
	UpdatedAt time.Time

	// Parsed from [{ts}]
	LogTime time.Time `gorm:"index:idx_log_time_status,priority:1;index:idx_log_time;not null"` // 复合索引和单独索引

	// GeoIP fields
	Country  string `gorm:"size:100;index"`
	Province string `gorm:"size:100"`
	City     string `gorm:"size:100"`

	// Request fields
	Host   string `gorm:"size:255;index:idx_host_log_time,priority:2"`                    // Host 和 LogTime 复合索引
	Method string `gorm:"size:10"`
	Uri    string `gorm:"type:text"`                                                       // URLs can be long
	Proto  string `gorm:"size:20"`
	Status int    `gorm:"index:idx_log_time_status,priority:2;index:idx_status_log_time,priority:1"` // 多个复合索引
	Size   int64

	// Client info
	UserAgent string `gorm:"type:text"`
	RemoteIP  string `gorm:"size:50;index:idx_remote_ip_log_time,priority:1"` // RemoteIP 和 LogTime 复合索引
	ClientIP  string `gorm:"size:50"`

	// 混合存储 - JSON 字段
	RawLog    string `gorm:"type:jsonb;comment:原始完整日志"`
	ExtraData string `gorm:"type:jsonb;comment:扩展元数据"`
}

// TableName 返回表名
func (CaddyLog) TableName() string {
	return "caddy_logs"
}

// CaddyLogArchive 归档日志表（结构与 CaddyLog 相同）
type CaddyLogArchive struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time

	LogTime time.Time `gorm:"index;not null"`

	Country  string `gorm:"size:100;index"`
	Province string `gorm:"size:100"`
	City     string `gorm:"size:100"`

	Host   string `gorm:"size:255;index"`
	Method string `gorm:"size:10"`
	Uri    string `gorm:"type:text"`
	Proto  string `gorm:"size:20"`
	Status int    `gorm:"index"`
	Size   int64

	UserAgent string `gorm:"type:text"`
	RemoteIP  string `gorm:"size:50;index"`
	ClientIP  string `gorm:"size:50"`

	RawLog    string `gorm:"type:jsonb;comment:原始完整日志"`
	ExtraData string `gorm:"type:jsonb;comment:扩展元数据"`
}

func (CaddyLogArchive) TableName() string {
	return "caddy_logs_archive"
}
