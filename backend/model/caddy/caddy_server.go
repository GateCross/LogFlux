package caddy

import (
	"time"
)

// LegacyServerID 是单实例改造后写进各表 server_id 列的固定值。
//
// 表结构保留 server_id（删列要迁移且历史要留），但语义收敛为「唯一实例」；
// 0 同时是证书库存查询「全部服务器」的既有取值，两者一致。
const LegacyServerID uint = 0

// CaddyServer 仅保存 Caddy Admin API 的连接信息。
//
// 单实例改造后配置不再存这里：Caddyfile 全文与 Modules 已迁到
// system_configs 的 caddy_config 键（见 config_store.go），本表只作为
// Admin API 地址与 Token 的载体，且永远只有一行。
type CaddyServer struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"uniqueIndex;not null"`
	Url       string `gorm:"not null"`
	Token     string
	Type      string `gorm:"default:'local'"` // 历史列，恒为 local
	Username  string
	Password  string
	// Config 与 Modules 为单实例改造前的配置存储列，仅作数据搬迁来源保留，
	// 运行时不再读写；配置真值在 system_configs 的 caddy_config 键。
	Config  string `gorm:"type:text"`
	Modules string `gorm:"type:jsonb"`
}

func (CaddyServer) TableName() string {
	return "caddy_servers"
}
