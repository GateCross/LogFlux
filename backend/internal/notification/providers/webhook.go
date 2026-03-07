package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"logflux/internal/notification"
	"logflux/model"
	"net/http"
	"strings"
	"time"
)

// WebhookProvider Webhook 通知提供者
type WebhookProvider struct {
	client *http.Client
}

// NewWebhookProvider 创建 Webhook 提供者
func NewWebhookProvider() *WebhookProvider {
	return &WebhookProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send 发送通知
func (w *WebhookProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	// 解析配置
	webhookConfig := &model.WebhookConfig{}
	if err := mapToStruct(config, webhookConfig); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}

	payload := buildWebhookPayload(webhookConfig, event)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 确定 HTTP 方法
	method := "POST"
	if webhookConfig.Method != "" {
		method = webhookConfig.Method
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, webhookConfig.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置默认 Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 设置自定义 Headers
	for key, value := range webhookConfig.Headers {
		req.Header.Set(key, value)
	}

	if webhookConfig.APIKey != "" {
		headerName := webhookConfig.APIKeyHeader
		if headerName == "" {
			headerName = "apiKey"
		}
		req.Header.Set(headerName, webhookConfig.APIKey)
	}

	// 发送请求
	resp, err := w.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("webhook request timed out: %w", err)
		}
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, _ := io.ReadAll(resp.Body)

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Validate 验证配置
func (w *WebhookProvider) Validate(config map[string]interface{}) error {
	webhookConfig := &model.WebhookConfig{}
	if err := mapToStruct(config, webhookConfig); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}

	return validateWebhookConfig(webhookConfig)
}

// Type 返回提供者类型
func (w *WebhookProvider) Type() string {
	return model.ChannelTypeWebhook
}

// validateWebhookConfig 验证 Webhook 配置
func validateWebhookConfig(config *model.WebhookConfig) error {
	if config.URL == "" {
		return fmt.Errorf("webhook url is required")
	}

	// 验证 URL 格式
	if !isValidURL(config.URL) {
		return fmt.Errorf("invalid webhook url: %s", config.URL)
	}

	// 验证 HTTP 方法
	if config.Method != "" {
		validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true}
		if !validMethods[config.Method] {
			return fmt.Errorf("invalid http method: %s", config.Method)
		}
	}

	if config.PayloadMode != "" {
		validModes := map[string]bool{"default": true, "message_api": true}
		if !validModes[config.PayloadMode] {
			return fmt.Errorf("invalid payload mode: %s", config.PayloadMode)
		}
	}

	for key := range config.BodyFields {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("body field key cannot be empty")
		}
	}

	return nil
}

func buildWebhookPayload(config *model.WebhookConfig, event *notification.Event) map[string]interface{} {
	if len(config.BodyFields) > 0 {
		return buildCustomWebhookPayload(config, event)
	}

	payloadMode := strings.TrimSpace(config.PayloadMode)
	if payloadMode == "" {
		payloadMode = "default"
	}

	switch payloadMode {
	case "message_api":
		titleField := strings.TrimSpace(config.TitleField)
		if titleField == "" {
			titleField = "title"
		}

		contentField := strings.TrimSpace(config.ContentField)
		if contentField == "" {
			contentField = "content"
		}

		return map[string]interface{}{
			titleField:   event.Title,
			contentField: resolveWebhookContent(event),
		}
	default:
		return map[string]interface{}{
			"type":      event.Type,
			"level":     event.Level,
			"title":     event.Title,
			"message":   event.Message,
			"data":      event.Data,
			"timestamp": event.Timestamp.Format(time.RFC3339),
		}
	}
}

func buildCustomWebhookPayload(config *model.WebhookConfig, event *notification.Event) map[string]interface{} {
	payload := make(map[string]interface{}, len(config.BodyFields))
	for key, valueType := range config.BodyFields {
		fieldName := strings.TrimSpace(key)
		if fieldName == "" {
			continue
		}

		switch strings.TrimSpace(valueType) {
		case "title":
			payload[fieldName] = event.Title
		case "content":
			payload[fieldName] = resolveWebhookContent(event)
		case "message":
			payload[fieldName] = event.Message
		case "level":
			payload[fieldName] = event.Level
		case "type":
			payload[fieldName] = event.Type
		case "timestamp":
			payload[fieldName] = event.Timestamp.Format(time.RFC3339)
		case "data":
			payload[fieldName] = event.Data
		default:
			payload[fieldName] = valueType
		}
	}
	return payload
}

func resolveWebhookContent(event *notification.Event) string {
	content := event.Message
	if event.Data != nil {
		if renderedContent, ok := event.Data["rendered_content"].(string); ok && strings.TrimSpace(renderedContent) != "" {
			content = renderedContent
		}
	}
	return content
}

// isValidURL 验证 URL 格式
func isValidURL(urlStr string) bool {
	// 简单验证: 必须以 http:// 或 https:// 开头
	return len(urlStr) > 8 && (urlStr[:7] == "http://" || urlStr[:8] == "https://")
}

// mapToStruct 将 map 转换为结构体
func mapToStruct(m map[string]interface{}, v interface{}) error {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, v)
}
