package providers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"logflux/internal/notification"
	"logflux/model"
	"strings"

	"gopkg.in/gomail.v2"
)

// EmailProvider 邮件通知提供者
type EmailProvider struct {
}

// NewEmailProvider 创建邮件提供者
func NewEmailProvider() *EmailProvider {
	return &EmailProvider{}
}

// Send 发送通知
func (e *EmailProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	// 解析配置
	emailConfig := &model.EmailConfig{}
	if err := mapToStruct(config, emailConfig); err != nil {
		return fmt.Errorf("invalid email config: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", emailConfig.From)
	m.SetHeader("To", emailConfig.To...)
	m.SetHeader("Subject", fmt.Sprintf("[%s] %s", strings.ToUpper(event.Level), event.Title))

	// 构建邮件正文
	var body string
	if content, ok := event.Data["rendered_content"]; ok && content != nil {
		if contentStr, ok := content.(string); ok {
			body = contentStr
		}
	}

	if body == "" {
		// Fallback: 使用默认格式
		body = fmt.Sprintf(`
		<h2>%s</h2>
		<p><strong>时间:</strong> %s</p>
		<p><strong>级别:</strong> %s</p>
		<p><strong>消息:</strong></p>
		<p>%s</p>
		<hr>
		<h3>详细信息:</h3>
		<pre>%s</pre>
	`, event.Title, event.Timestamp.Format("2006-01-02 15:04:05"), event.Level, event.Message, prettyJSON(event.Data))
	}

	m.SetBody("text/html", body)

	d := gomail.NewDialer(emailConfig.SmtpHost, emailConfig.SmtpPort, emailConfig.Username, emailConfig.Password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true} // 允许自签名证书 (可选)

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// Validate 验证配置
func (e *EmailProvider) Validate(config map[string]interface{}) error {
	emailConfig := &model.EmailConfig{}
	if err := mapToStruct(config, emailConfig); err != nil {
		return fmt.Errorf("invalid email config: %w", err)
	}

	return validateEmailConfig(emailConfig)
}

// Type 返回提供者类型
func (e *EmailProvider) Type() string {
	return model.ChannelTypeEmail
}

// validateEmailConfig 验证邮件配置
func validateEmailConfig(config *model.EmailConfig) error {
	if config.SmtpHost == "" {
		return fmt.Errorf("smtp_host is required")
	}
	if config.SmtpPort <= 0 {
		return fmt.Errorf("invalid smtp_port: %d", config.SmtpPort)
	}
	if config.Username == "" {
		return fmt.Errorf("username is required")
	}
	if config.Password == "" {
		return fmt.Errorf("password is required")
	}
	if config.From == "" {
		return fmt.Errorf("from address is required")
	}
	if len(config.To) == 0 {
		return fmt.Errorf("at least one recipient (to) is required")
	}
	return nil
}

// prettyJSON 格式化 JSON
func prettyJSON(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
