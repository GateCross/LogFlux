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
	"net/url"
	"strings"
	"sync"
	"time"
)

// WeChatMPProvider 微信公众号通知提供者（客服消息）
type WeChatMPProvider struct {
	client *http.Client

	mu          sync.Mutex
	cachedToken string
	tokenExpire time.Time
}

// NewWeChatMPProvider 创建微信公众号通知提供者
func NewWeChatMPProvider() *WeChatMPProvider {
	return &WeChatMPProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send 发送通知
func (w *WeChatMPProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	wechatConfig := &model.WechatMPConfig{}
	if err := mapToStruct(config, wechatConfig); err != nil {
		return fmt.Errorf("invalid wechat_mp config: %w", err)
	}

	if err := validateWeChatMPConfig(wechatConfig); err != nil {
		return err
	}

	accessToken, err := w.getAccessToken(ctx, wechatConfig)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"touser":  wechatConfig.ToUser,
		"msgtype": "text",
		"text": map[string]string{
			"content": formatWeChatMPMessage(event),
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal wechat_mp payload: %w", err)
	}

	sendURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s", url.QueryEscape(accessToken))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to wechat mp: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("wechat mp returned status: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse wechat mp response: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wechat mp api error: errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// Validate 验证配置
func (w *WeChatMPProvider) Validate(config map[string]interface{}) error {
	wechatConfig := &model.WechatMPConfig{}
	if err := mapToStruct(config, wechatConfig); err != nil {
		return fmt.Errorf("invalid wechat_mp config: %w", err)
	}

	return validateWeChatMPConfig(wechatConfig)
}

// Type 返回提供者类型
func (w *WeChatMPProvider) Type() string {
	return model.ChannelTypeWeChatMP
}

func validateWeChatMPConfig(config *model.WechatMPConfig) error {
	if strings.TrimSpace(config.AppID) == "" {
		return fmt.Errorf("app_id is required")
	}
	if strings.TrimSpace(config.AppSecret) == "" {
		return fmt.Errorf("app_secret is required")
	}
	if strings.TrimSpace(config.ToUser) == "" {
		return fmt.Errorf("to_user is required")
	}

	return nil
}

func (w *WeChatMPProvider) getAccessToken(ctx context.Context, config *model.WechatMPConfig) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cachedToken != "" && time.Now().Before(w.tokenExpire) {
		return w.cachedToken, nil
	}

	tokenURL := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		url.QueryEscape(config.AppID),
		url.QueryEscape(config.AppSecret),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get wechat mp access token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("wechat mp token api status: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return "", fmt.Errorf("wechat mp token api error: errcode=%d errmsg=%s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("wechat mp access token is empty")
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	if expiresIn <= 300 {
		expiresIn = 600
	}

	w.cachedToken = tokenResp.AccessToken
	// 留 5 分钟余量避免临界过期
	w.tokenExpire = time.Now().Add(time.Duration(expiresIn-300) * time.Second)

	return w.cachedToken, nil
}

func formatWeChatMPMessage(event *notification.Event) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[%s] %s\n", strings.ToUpper(event.Level), event.Title))
	builder.WriteString(event.Message)
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("事件: %s\n", event.Type))
	builder.WriteString(fmt.Sprintf("时间: %s", event.Timestamp.Format(time.DateTime)))

	if renderedContent, ok := event.Data["rendered_content"].(string); ok && strings.TrimSpace(renderedContent) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(renderedContent)
	}

	return builder.String()
}
