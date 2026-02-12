package notification

import (
	"time"
)

// Event 通知事件
type Event struct {
	// 事件类型 (如: system.startup, log.high_error_rate)
	Type string

	// 事件级别: info, warning, error, critical
	Level string

	// 事件标题
	Title string

	// 事件消息
	Message string

	// 事件数据 (自定义字段)
	Data map[string]interface{}

	// 事件时间
	Timestamp time.Time
}

// NewEvent 创建新事件
func NewEvent(eventType, level, title, message string) *Event {
	return &Event{
		Type:      eventType,
		Level:     level,
		Title:     title,
		Message:   message,
		Data:      make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// WithData 添加事件数据
func (e *Event) WithData(key string, value interface{}) *Event {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

// WithDataMap 批量添加事件数据
func (e *Event) WithDataMap(data map[string]interface{}) *Event {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	for k, v := range data {
		e.Data[k] = v
	}
	return e
}

// EventLevel 事件级别常量
const (
	LevelInfo     = "info"
	LevelWarning  = "warning"
	LevelError    = "error"
	LevelCritical = "critical"
)

// EventType 常用事件类型
const (
	// 系统事件
	EventSystemStartup            = "system.startup"
	EventSystemShutdown           = "system.shutdown"
	EventSystemError              = "system.error"
	EventRedisConnectionFailed    = "redis.connection_failed"
	EventDatabaseConnectionFailed = "database.connection_failed"

	// 日志采集事件
	EventLogParseError        = "log.parse_error"
	EventLogIngestFailed      = "log.ingest_failed"
	EventLogHighErrorRate     = "log.high_error_rate"
	EventLogSuspiciousIP      = "log.suspicious_ip"
	EventLogCollectionStopped = "log.collection_stopped"

	// 归档事件
	EventArchiveFailed    = "archive.failed"
	EventArchiveCompleted = "archive.completed"
	EventArchiveSlow      = "archive.slow"
	EventArchiveAnomaly   = "archive.anomaly"

	// Caddy 配置事件
	EventCaddyConfigUpdateFailed  = "caddy.config_update_failed"
	EventCaddyConfigUpdateSuccess = "caddy.config_update_success"
	EventCaddyLogSourceDiscovered = "caddy.log_source_discovered"

	// 安全事件
	EventSecurityLoginFailed             = "security.login_failed"
	EventSecurityBruteForce              = "security.brute_force"
	EventSecurityAdminLogin              = "security.admin_login"
	EventSecurityPermissionDenied        = "security.permission_denied"
	EventSecurityWafPolicyPublishFailed  = "security.waf_policy_publish_failed"
	EventSecurityWafPolicyPublished      = "security.waf_policy_published"
	EventSecurityWafPolicyRollback       = "security.waf_policy_rollback"
	EventSecurityWafPolicyRollbackFailed = "security.waf_policy_rollback_failed"
	EventSecurityWafPolicyAutoRollback   = "security.waf_policy_auto_rollback"
)
