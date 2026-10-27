package caddy

import (
	"time"
)

// LegacyServerID 是单实例改造后写进各表 server_id 列的固定值。
//
// 表结构保留 server_id（删列要迁移且历史要留），但语义收敛为「唯一实例」；
// 0 同时是证书库存查询「全部服务器」的既有取值，两者一致。
const LegacyServerID uint = 0

// CaddyServer 只保存 Caddy Admin API 的连接信息，且永远只有一行。
//
// 配置真值在 system_configs 的 caddy_config 键（见 config_store.go），不在这张表。
// 本表由启动播种维护，不再有增删改入口。
type CaddyServer struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"uniqueIndex;not null"`
	Url       string `gorm:"not null"`
	Token     string
	// 以下列是单实例改造前的历史遗留，运行时不再读写，保留只为老库迁移不报错。
	Type     string `gorm:"default:'local'"`
	Username string
	Password string
	Config   string `gorm:"type:text"`
	Modules  string `gorm:"type:jsonb"`
}

func (CaddyServer) TableName() string {
	return "caddy_servers"
}
