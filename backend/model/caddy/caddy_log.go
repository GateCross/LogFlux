package caddy

import (
	"time"
)

type CaddyLog struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"` // Index for time-range queries
	UpdatedAt time.Time

	// Parsed from [{ts}]
	// LogTime 同时参与 idx_host_log_time 的复合索引（priority:1），
	// 缺了这一处声明，idx_host_log_time 会退化成只有 host 的单列索引。
	LogTime time.Time `gorm:"index:idx_log_time_status,priority:1;index:idx_log_time;index:idx_host_log_time,priority:1;not null"`

	// GeoIP fields
	Country  string `gorm:"size:100;index"`
	Province string `gorm:"size:100"`
	City     string `gorm:"size:100"`

	// 日志级别与 logger 命名空间，取自 Caddy JSON 的 level / logger 字段。
	// 访问日志为 logger=http.log.access；错误日志来自其它命名空间（http.log、tls 等）。
	// 这两个字段是「访问日志 / 错误日志分开查看」的唯一数据依据。
	Level  string `gorm:"size:16;index:idx_level_log_time,priority:1"`
	Logger string `gorm:"size:128;index:idx_logger_log_time,priority:1"`

	// Request fields
	Host   string `gorm:"size:255;index:idx_host_log_time,priority:2"` // Host 和 LogTime 复合索引
	Method string `gorm:"size:10"`
	Uri    string `gorm:"type:text"` // URLs can be long
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
