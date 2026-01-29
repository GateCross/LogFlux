package providers

import (
	"context"
	"fmt"
	"logflux/internal/notification"
	"logflux/model"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramProvider Telegram 通知提供者
type TelegramProvider struct{}

// NewTelegramProvider 创建 Telegram 提供者
func NewTelegramProvider() *TelegramProvider {
	return &TelegramProvider{}
}

// Send 发送通知
func (t *TelegramProvider) Send(ctx context.Context, config map[string]interface{}, event *notification.Event) error {
	// 解析配置
	tgConfig := &model.TelegramConfig{}
	if err := mapToStruct(config, tgConfig); err != nil {
		return fmt.Errorf("invalid telegram config: %w", err)
	}

	// 创建 bot
	bot, err := tgbotapi.NewBotAPI(tgConfig.BotToken)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}

	// 解析 chat_id
	chatID, err := strconv.ParseInt(tgConfig.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}

	// 构建 Markdown 格式消息
	message := formatTelegramMessage(event)

	// 创建消息
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "MarkdownV2"

	// 发送消息
	_, err = bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	return nil
}

// Validate 验证配置
func (t *TelegramProvider) Validate(config map[string]interface{}) error {
	tgConfig := &model.TelegramConfig{}
	if err := mapToStruct(config, tgConfig); err != nil {
		return fmt.Errorf("invalid telegram config: %w", err)
	}

	return validateTelegramConfig(tgConfig)
}

// Type 返回提供者类型
func (t *TelegramProvider) Type() string {
	return model.ChannelTypeTelegram
}

// validateTelegramConfig 验证 Telegram 配置
func validateTelegramConfig(config *model.TelegramConfig) error {
	if config.BotToken == "" {
		return fmt.Errorf("bot_token is required")
	}
	if config.ChatID == "" {
		return fmt.Errorf("chat_id is required")
	}

	// 验证 chat_id 格式
	if _, err := strconv.ParseInt(config.ChatID, 10, 64); err != nil {
		return fmt.Errorf("invalid chat_id format: must be a number")
	}

	return nil
}

// formatTelegramMessage 格式化 Telegram 消息 (Markdown V2)
func formatTelegramMessage(event *notification.Event) string {
	var builder strings.Builder

	// 标题 (加粗)
	builder.WriteString("*")
	builder.WriteString(escapeMD(event.Title))
	builder.WriteString("*\n\n")

	// 级别图标
	levelIcon := getLevelIcon(event.Level)
	builder.WriteString(levelIcon)
	builder.WriteString(" *级别:* ")
	builder.WriteString(escapeMD(event.Level))
	builder.WriteString("\n")

	// 时间
	builder.WriteString("🕒 *时间:* ")
	builder.WriteString(escapeMD(event.Timestamp.Format("2006-01-02 15:04:05")))
	builder.WriteString("\n\n")

	// 消息内容
	builder.WriteString("📝 *消息:*\n")
	builder.WriteString(escapeMD(event.Message))
	builder.WriteString("\n")

	// 详细数据 (如果存在)
	if event.Data != nil && len(event.Data) > 0 {
		builder.WriteString("\n📊 *详细信息:*\n")
		builder.WriteString("```json\n")
		builder.WriteString(prettyJSON(event.Data))
		builder.WriteString("\n```")
	}

	return builder.String()
}

// getLevelIcon 获取级别对应的图标
func getLevelIcon(level string) string {
	icons := map[string]string{
		"info":     "ℹ️",
		"warning":  "⚠️",
		"error":    "❌",
		"critical": "🚨",
		"success":  "✅",
	}

	if icon, ok := icons[level]; ok {
		return icon
	}
	return "📌"
}

// escapeMD 转义 Markdown V2 特殊字符
func escapeMD(text string) string {
	// Telegram Markdown V2 需要转义的字符
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

	result := text
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}
