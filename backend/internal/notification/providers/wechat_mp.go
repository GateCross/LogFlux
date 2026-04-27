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

// WeChatMPProvider 企业微信应用消息提供者
type WeChatMPProvider struct {
	client *http.Client

	mu         sync.Mutex
	tokenCache map[string]tokenCacheItem
}

type tokenCacheItem struct {
	token    string
	expireAt time.Time
}

// NewWeChatMPProvider 创建企业微信应用消息提供者
func NewWeChatMPProvider() *WeChatMPProvider {
	return &WeChatMPProvider{
		client:     &http.Client{Timeout: 10 * time.Second},
		tokenCache: make(map[string]tokenCacheItem),
	}
}

// Send 发送通知
func (w *WeChatMPProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	wechatConfig := &model.WechatMPConfig{}
	if err := mapToStruct(config, wechatConfig); err != nil {
		return fmt.Errorf("企业微信应用配置无效: %w", err)
	}

	if err := validateWeChatMPConfig(wechatConfig); err != nil {
		return err
	}

	accessToken, err := w.getAccessToken(ctx, wechatConfig)
	if err != nil {
		return err
	}

	msgType := strings.ToLower(strings.TrimSpace(wechatConfig.MsgType))
	if msgType == "" {
		msgType = "text"
	}

	touser := strings.TrimSpace(wechatConfig.ToUser)
	toparty := strings.TrimSpace(wechatConfig.ToParty)
	totag := strings.TrimSpace(wechatConfig.ToTag)
	if touser == "" && toparty == "" && totag == "" {
		touser = "@all"
	}

	payload := map[string]interface{}{
		"msgtype": msgType,
		"agentid": wechatConfig.AgentID,
		"safe":    0,
	}
	if touser != "" {
		payload["touser"] = touser
	}
	if toparty != "" {
		payload["toparty"] = toparty
	}
	if totag != "" {
		payload["totag"] = totag
	}

	if msgType == "markdown" {
		payload["markdown"] = map[string]string{
			"content": formatWeComAppMarkdownMessage(event),
		}
	} else {
		payload["msgtype"] = "text"
		payload["text"] = map[string]string{
			"content": formatWeComAppTextMessage(event),
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化企业微信应用消息失败: %w", err)
	}

	sendURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", url.QueryEscape(accessToken))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送企业微信应用请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("企业微信应用返回状态: %d，响应: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析企业微信应用响应失败: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("企业微信应用 API 错误: errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// Validate 验证配置
func (w *WeChatMPProvider) Validate(config map[string]interface{}) error {
	wechatConfig := &model.WechatMPConfig{}
	if err := mapToStruct(config, wechatConfig); err != nil {
		return fmt.Errorf("企业微信应用配置无效: %w", err)
	}

	return validateWeChatMPConfig(wechatConfig)
}

// Type 返回提供者类型
func (w *WeChatMPProvider) Type() string {
	return model.ChannelTypeWeChatMP
}

func validateWeChatMPConfig(config *model.WechatMPConfig) error {
	if strings.TrimSpace(config.CorpID) == "" {
		return fmt.Errorf("企业 ID 不能为空")
	}
	if strings.TrimSpace(config.CorpSecret) == "" {
		return fmt.Errorf("企业密钥不能为空")
	}
	if config.AgentID <= 0 {
		return fmt.Errorf("应用 AgentID 必须大于 0")
	}

	msgType := strings.ToLower(strings.TrimSpace(config.MsgType))
	if msgType != "" && msgType != "text" && msgType != "markdown" {
		return fmt.Errorf("消息类型必须是 text 或 markdown")
	}

	return nil
}

func (w *WeChatMPProvider) getAccessToken(ctx context.Context, config *model.WechatMPConfig) (string, error) {
	cacheKey := fmt.Sprintf("%s|%s", config.CorpID, config.CorpSecret)

	w.mu.Lock()
	if item, ok := w.tokenCache[cacheKey]; ok && item.token != "" && time.Now().Before(item.expireAt) {
		token := item.token
		w.mu.Unlock()
		return token, nil
	}
	w.mu.Unlock()

	tokenURL := fmt.Sprintf(
		"https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		url.QueryEscape(config.CorpID),
		url.QueryEscape(config.CorpSecret),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建令牌请求失败: %w", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取企业微信应用 access_token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("企业微信应用令牌接口返回状态: %d，响应: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("解析令牌响应失败: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return "", fmt.Errorf("企业微信应用令牌接口错误: errcode=%d errmsg=%s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("企业微信应用 access_token 为空")
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	if expiresIn <= 300 {
		expiresIn = 600
	}

	token := tokenResp.AccessToken
	// 留 5 分钟余量避免临界过期
	expireAt := time.Now().Add(time.Duration(expiresIn-300) * time.Second)

	w.mu.Lock()
	w.tokenCache[cacheKey] = tokenCacheItem{token: token, expireAt: expireAt}
	w.mu.Unlock()

	return token, nil
}

func formatWeComAppTextMessage(event *notification.Event) string {
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

func formatWeComAppMarkdownMessage(event *notification.Event) string {
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
