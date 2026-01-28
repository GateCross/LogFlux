package notification

import (
	"context"
)

// NotificationManager 通知管理器接口
type NotificationManager interface {
	// Notify 发送通知
	// 根据事件类型匹配通知渠道并发送
	Notify(ctx context.Context, event *Event) error

	// RegisterProvider 注册通知提供者
	RegisterProvider(provider NotificationProvider) error

	// EvaluateRules 评估规则并触发通知
	// 根据数据评估所有规则,触发符合条件的告警
	EvaluateRules(ctx context.Context, data map[string]interface{}) error

	// Start 启动通知管理器
	// 加载配置、初始化提供者等
	Start(ctx context.Context) error

	// Stop 停止通知管理器
	// 清理资源、关闭连接等
	Stop() error

	// ReloadChannels 重新加载通知渠道配置
	ReloadChannels() error

	// ReloadRules 重新加载告警规则
	ReloadRules() error
}
