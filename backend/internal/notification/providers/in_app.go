package providers

import (
	"context"
	"logflux/internal/notification"
)

// InAppProvider 应用内通知提供者
type InAppProvider struct{}

// NewInAppProvider 创建应用内通知提供者
func NewInAppProvider() *InAppProvider {
	return &InAppProvider{}
}

// Type 返回提供者类型
func (p *InAppProvider) Type() string {
	return "in_app"
}

// Send 发送通知 (实际上只是确认接收，由 Manager 负责写入数据库)
func (p *InAppProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	// 应用内通知不需要实际发送，只需要 Manager 将日志写入数据库即可
	// Manager 会在 Send 返回 nil 后将日志状态更新为 Success
	return nil
}

// Validate 验证配置
func (p *InAppProvider) Validate(config map[string]interface{}) error {
	return nil
}
