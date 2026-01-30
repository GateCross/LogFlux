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

// DiscordProvider Discord 通知提供者
type DiscordProvider struct {
	client *http.Client
}

// NewDiscordProvider 创建 Discord 提供者
func NewDiscordProvider() *DiscordProvider {
	return &DiscordProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send 发送通知
func (d *DiscordProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	// 解析配置
	discordConfig := &model.DiscordConfig{}
	if err := mapToStruct(config, discordConfig); err != nil {
		return fmt.Errorf("invalid discord config: %w", err)
	}

	// 构建消息
	// Discord Webhook Format: https://discord.com/developers/docs/resources/webhook#execute-webhook
	message := map[string]interface{}{
		"content": fmt.Sprintf("**[%s] %s**\n%s", event.Level, event.Title, event.Message),
	}

	if discordConfig.Username != "" {
		message["username"] = discordConfig.Username
	}
	if discordConfig.AvatarURL != "" {
		message["avatar_url"] = discordConfig.AvatarURL
	}

	// 如果有详细数据，作为 Embed 发送
	if len(event.Data) > 0 {
		dataBytes, _ := json.MarshalIndent(event.Data, "", "  ")

		color := 0x3498db // Default Blue
		switch event.Level {
		case notification.LevelError, notification.LevelCritical:
			color = 0xe74c3c // Red
		case notification.LevelWarning:
			color = 0xf1c40f // Yellow
		case "success": // Custom string literal since constant doesn't exist
			color = 0x2ecc71 // Green
		}

		message["embeds"] = []map[string]interface{}{
			{
				"title":       "Metadata",
				"description": fmt.Sprintf("```json\n%s\n```", string(dataBytes)),
				"color":       color,
				"timestamp":   event.Timestamp.Format(time.RFC3339),
			},
		}
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal discord message: %w", err)
	}

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "POST", discordConfig.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to discord: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

// Validate 验证配置
func (d *DiscordProvider) Validate(config map[string]interface{}) error {
	discordConfig := &model.DiscordConfig{}
	if err := mapToStruct(config, discordConfig); err != nil {
		return fmt.Errorf("invalid discord config: %w", err)
	}

	if discordConfig.WebhookURL == "" {
		return fmt.Errorf("webhook url is required")
	}

	return nil
}

// Type 返回提供者类型
func (d *DiscordProvider) Type() string {
	return model.ChannelTypeDiscord
}
