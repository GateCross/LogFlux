package ingest

import (
	"time"
)

type LogSource struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name string `gorm:"size:255;not null"`
	// Path 与 ServerID 共同唯一：不同节点可能配置同名日志路径
	// （如都用 /var/log/caddy/access.log），只按路径唯一会互相覆盖。
	Path         string `gorm:"size:1024;not null;uniqueIndex:uk_log_source_server_path,priority:1"`
	Type         string `gorm:"size:50;default:'caddy'"` // Source type (caddy, nginx, etc)
	Enabled      bool   `gorm:"default:true"`            // Is monitoring active?
	ScanInterval int    `gorm:"default:60"`              // Directory scan interval (seconds)
	// ServerID 标记日志源所属的 Caddy 节点。自动发现的源按节点隔离同步，
	// 避免一个节点保存配置时把其它节点的自动源误禁用。
	// 0 表示未绑定节点（手工注册的全局源，如 WAF 审计日志）。
	ServerID uint `gorm:"not null;default:0;uniqueIndex:uk_log_source_server_path,priority:2"`
}

func (LogSource) TableName() string {
	return "log_sources"
}
