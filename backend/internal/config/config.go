package config

import (
	"fmt"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
		AccessExpire int64
		AESKey       string
	}
	Database            DatabaseConf
	Redis               RedisConf
	CaddyLogPath        string `json:",optional"`
	BackendLogPath      string `json:",optional"` // 后端日志文件/目录（用于入库）
	CaddyRuntimeLogPath string `json:",optional"` // Caddy 后台日志文件/目录（用于入库）
	Archive             ArchiveConf
	WAF                 WAFConf
	Notification        NotificationConf `json:",optional"`
}

type DatabaseConf struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (c DatabaseConf) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}

type RedisConf struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (c RedisConf) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type ArchiveConf struct {
	Enabled      bool
	RetentionDay int // 日志保留天数
	ArchiveTable string
}

type WAFConf struct {
	WorkDir              string
	MaxPackageBytes      int64
	AllowedDomains       []string
	ExtractMaxFiles      int
	ExtractMaxTotalBytes int64
	ActivateTimeoutSec   int
}

// NotificationConf 通知配置
type NotificationConf struct {
	Enabled         bool          // 是否启用通知功能
	DefaultChannels []string      // 默认通知渠道名称列表
	Channels        []ChannelConf // 通知渠道配置
	Rules           []RuleConf    // 告警规则配置
}

// ChannelConf 通知渠道配置
type ChannelConf struct {
	Name        string                 // 渠道名称
	Type        string                 // 渠道类型: webhook, email, telegram, slack, wecom, wechat_mp, dingtalk
	Enabled     bool                   // 是否启用
	Config      map[string]interface{} // 渠道配置 (根据类型不同而不同)
	Events      []string               // 订阅的事件类型 (支持通配符)
	Description string                 // 描述
}

// RuleConf 告警规则配置
type RuleConf struct {
	Name            string                 // 规则名称
	Enabled         bool                   // 是否启用
	RuleType        string                 // 规则类型: threshold, frequency, ratio, pattern, composite
	EventType       string                 // 触发事件类型
	Condition       map[string]interface{} // 条件表达式
	ChannelNames    []string               // 通知渠道名称列表 (而不是 ID)
	Template        string                 `json:",optional"` // 自定义模板 (可选)
	SilenceDuration int                    // 静默时间 (秒)
	Description     string                 // 描述
}
