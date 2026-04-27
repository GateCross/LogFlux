package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"logflux/internal/notification"
	"logflux/model"
	"net/http"
	"strings"
	"time"
)

// WeComProvider 企业微信机器人通知提供者
type WeComProvider struct {
	client *http.Client
}

// NewWeComProvider 创建企业微信通知提供者
func NewWeComProvider() *WeComProvider {
	return &WeComProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send 发送通知
func (w *WeComProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	wecomConfig := &model.WeComConfig{}
	if err := mapToStruct(config, wecomConfig); err != nil {
		return fmt.Errorf("企业微信配置无效: %w", err)
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": formatWeComMessage(event),
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化企业微信消息失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wecomConfig.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送企业微信请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("企业微信 Webhook 返回状态: %d，响应: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &result); err == nil && result.ErrCode != 0 {
			return fmt.Errorf("企业微信 API 错误: errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
		}
	}

	return nil
}

// Validate 验证配置
func (w *WeComProvider) Validate(config map[string]interface{}) error {
	wecomConfig := &model.WeComConfig{}
	if err := mapToStruct(config, wecomConfig); err != nil {
		return fmt.Errorf("企业微信配置无效: %w", err)
	}

	if strings.TrimSpace(wecomConfig.WebhookURL) == "" {
		return fmt.Errorf("回调 URL 不能为空")
	}

	if !isValidURL(wecomConfig.WebhookURL) {
		return fmt.Errorf("回调 URL 无效: %s", wecomConfig.WebhookURL)
	}

	return nil
}

// Type 返回提供者类型
func (w *WeComProvider) Type() string {
	return model.ChannelTypeWeCom
}

func formatWeComMessage(event *notification.Event) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("### [%s] %s\n", strings.ToUpper(event.Level), event.Title))
	builder.WriteString(event.Message)
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("> 事件: %s\n", event.Type))
	builder.WriteString(fmt.Sprintf("> 时间: %s", event.Timestamp.Format(time.DateTime)))

	if renderedContent, ok := event.Data["rendered_content"].(string); ok && strings.TrimSpace(renderedContent) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(renderedContent)
	}

	return builder.String()
}
