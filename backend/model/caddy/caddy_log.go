package caddy

import (
	"strings"
	"time"

	"gorm.io/gorm"
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
	// logger 是「访问日志 / 错误日志分开查看」的唯一分类依据；
	// level 只是严重程度，两者正交，不要把 level 混进种类判断。
	// 空值由 BeforeCreate 兜底成 info / http.log.access。
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

// 访问日志的 logger 命名空间根名。
// 站点给 log 指令写了名字时是 http.log.access.<名字>，同一 Caddyfile 里有多个站点时
// Caddy 按书写顺序生成 http.log.access.log0、log1…，所以判定必须按前缀，不能等值比较。
const caddyAccessLoggerName = "http.log.access"

// BeforeCreate 把 level / logger 归一化后再入库。
//
// 采集侧有 4 条写入路径（JSON 访问日志、Coraza 审计、console 文本、面板中间件），
// 都走 GORM 的 Create，所以这里兜一次就全覆盖，各路径不必各自记得填。
// 规则与 log_field_backfill.go 对存量行的回填一致：空 logger 归访问日志，
// 空 level 按 info 处理（Caddy 访问日志本身也只产出 INFO 与 ERROR 两级）。
func (l *CaddyLog) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(l.Logger) == "" {
		l.Logger = caddyAccessLoggerName
	}
	if strings.TrimSpace(l.Level) == "" {
		l.Level = "info"
	}
	return nil
}
