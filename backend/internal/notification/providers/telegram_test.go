package providers

import (
	"logflux/internal/notification"
	"testing"
	"time"
)

func TestTelegramProvider_Validate(t *testing.T) {
	provider := NewTelegramProvider()

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"bot_token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
				"chat_id":   "123456789",
			},
			wantErr: false,
		},
		{
			name: "missing bot_token",
			config: map[string]interface{}{
				"chat_id": "123456789",
			},
			wantErr: true,
			errMsg:  "Bot Token 不能为空",
		},
		{
			name: "missing chat_id",
			config: map[string]interface{}{
				"bot_token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
			},
			wantErr: true,
			errMsg:  "Chat ID 不能为空",
		},
		{
			name: "Chat ID 格式无效",
			config: map[string]interface{}{
				"bot_token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
				"chat_id":   "not-a-number",
			},
			wantErr: true,
			errMsg:  "Chat ID 格式无效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestTelegramProvider_Type(t *testing.T) {
	provider := NewTelegramProvider()
	if got := provider.Type(); got != "telegram" {
		t.Errorf("Type() = %v, want %v", got, "telegram")
	}
}

func TestFormatTelegramMessage(t *testing.T) {
	event := &notification.Event{
		Type:      "system.test",
		Level:     "info",
		Title:     "Test Event",
		Message:   "This is a test message",
		Timestamp: time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC),
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	message := formatTelegramMessage(event)

	// 验证消息包含关键信息
	if !contains(message, "Test Event") {
		t.Error("Message should contain title")
	}
	if !contains(message, "info") {
		t.Error("Message should contain level")
	}
	if !contains(message, "This is a test message") {
		t.Error("Message should contain message content")
	}
	// 时间戳中的 "-" 会被转义,所以检查年份即可
	if !contains(message, "2026") {
		t.Error("Message should contain timestamp year")
	}
}

func TestGetLevelIcon(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"info", "ℹ️"},
		{"warning", "⚠️"},
		{"error", "❌"},
		{"critical", "🚨"},
		{"success", "✅"},
		{"unknown", "📌"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			if got := getLevelIcon(tt.level); got != tt.want {
				t.Errorf("getLevelIcon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeMD(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello_world", "hello\\_world"},
		{"test*bold*", "test\\*bold\\*"},
		{"[link](url)", "\\[link\\]\\(url\\)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := escapeMD(tt.input); got != tt.want {
				t.Errorf("escapeMD() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && anySubstring(s, substr))
}

func anySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
