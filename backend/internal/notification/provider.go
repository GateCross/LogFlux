package notification

import (
	"context"
)

// NotificationProvider 通知提供者接口
type NotificationProvider interface {
	// Send 发送通知
	Send(ctx context.Context, config map[string]interface{}, event *Event) error

	// Validate 验证配置是否正确
	Validate(config map[string]interface{}) error

	// Type 返回提供者类型
	Type() string
}
