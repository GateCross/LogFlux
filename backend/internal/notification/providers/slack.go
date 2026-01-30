package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"logflux/internal/notification"
	"logflux/model"
	"net/http"
	"time"
)

// SlackProvider Slack 通知提供者
type SlackProvider struct {
	client *http.Client
}

// NewSlackProvider 创建 Slack 提供者
func NewSlackProvider() *SlackProvider {
	return &SlackProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send 发送通知
func (s *SlackProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	// 解析配置
	slackConfig := &model.SlackConfig{}
	if err := mapToStruct(config, slackConfig); err != nil {
		return fmt.Errorf("invalid slack config: %w", err)
	}

	// 构建消息
	// Slack Message Format: https://api.slack.com/messaging/webhooks
	message := map[string]interface{}{
		"text": fmt.Sprintf("*[%s] %s*\n%s", event.Level, event.Title, event.Message),
	}

	// 如果有详细数据，作为附件或 Block 发送 (这里简化处理)
	if len(event.Data) > 0 {
		dataBytes, _ := json.MarshalIndent(event.Data, "", "  ")
		message["attachments"] = []map[string]interface{}{
			{
				"text": fmt.Sprintf("```%s```", string(dataBytes)),
			},
		}
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "POST", slackConfig.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

// Validate 验证配置
func (s *SlackProvider) Validate(config map[string]interface{}) error {
	slackConfig := &model.SlackConfig{}
	if err := mapToStruct(config, slackConfig); err != nil {
		return fmt.Errorf("invalid slack config: %w", err)
	}

	if slackConfig.WebhookURL == "" {
		return fmt.Errorf("webhook url is required")
	}

	return nil
}

// Type 返回提供者类型
func (s *SlackProvider) Type() string {
	return model.ChannelTypeSlack
}
