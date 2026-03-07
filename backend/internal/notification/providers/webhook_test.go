package providers

import (
	"context"
	"encoding/json"
	"logflux/internal/notification"
	"logflux/model"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookProviderValidate(t *testing.T) {
	provider := NewWebhookProvider()

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid message api config",
			config: map[string]interface{}{
				"url":            "https://example.com/api/v1/message/openSend",
				"method":         "POST",
				"payload_mode":   "message_api",
				"api_key":        "test-key",
				"api_key_header": "apiKey",
			},
			wantErr: false,
		},
		{
			name: "invalid payload mode",
			config: map[string]interface{}{
				"url":          "https://example.com/hook",
				"payload_mode": "custom",
			},
			wantErr: true,
		},
		{
			name: "missing url",
			config: map[string]interface{}{
				"payload_mode": "message_api",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := provider.Validate(testCase.config)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestBuildWebhookPayloadMessageAPI(t *testing.T) {
	event := &notification.Event{
		Type:      "system.test",
		Level:     notification.LevelInfo,
		Title:     "Test Title",
		Message:   "Fallback Message",
		Timestamp: time.Unix(1700000000, 0),
		Data: map[string]interface{}{
			"rendered_content": "Rendered Content",
		},
	}

	payload := buildWebhookPayload(&model.WebhookConfig{
		PayloadMode:  "message_api",
		TitleField:   "title",
		ContentField: "content",
	}, event)

	if got := payload["title"]; got != "Test Title" {
		t.Fatalf("payload title = %v, want %v", got, "Test Title")
	}

	if got := payload["content"]; got != "Rendered Content" {
		t.Fatalf("payload content = %v, want %v", got, "Rendered Content")
	}
}

func TestBuildWebhookPayloadCustomFields(t *testing.T) {
	event := &notification.Event{
		Type:      "system.test",
		Level:     notification.LevelInfo,
		Title:     "Test Title",
		Message:   "Fallback Message",
		Timestamp: time.Unix(1700000000, 0),
		Data: map[string]interface{}{
			"rendered_content": "Rendered Content",
		},
	}

	payload := buildWebhookPayload(&model.WebhookConfig{
		BodyFields: map[string]string{
			"title":       "title",
			"content":     "content",
			"msgType":     "text",
			"event_level": "level",
		},
	}, event)

	if got := payload["title"]; got != "Test Title" {
		t.Fatalf("payload title = %v, want %v", got, "Test Title")
	}

	if got := payload["content"]; got != "Rendered Content" {
		t.Fatalf("payload content = %v, want %v", got, "Rendered Content")
	}

	if got := payload["msgType"]; got != "text" {
		t.Fatalf("payload msgType = %v, want %v", got, "text")
	}

	if got := payload["event_level"]; got != notification.LevelInfo {
		t.Fatalf("payload event_level = %v, want %v", got, notification.LevelInfo)
	}
}

func TestBuildWebhookPayloadDefault(t *testing.T) {
	event := &notification.Event{
		Type:      "system.test",
		Level:     notification.LevelInfo,
		Title:     "Test Title",
		Message:   "Fallback Message",
		Timestamp: time.Unix(1700000000, 0),
		Data:      map[string]interface{}{"foo": "bar"},
	}

	payload := buildWebhookPayload(&model.WebhookConfig{}, event)

	if got := payload["title"]; got != "Test Title" {
		t.Fatalf("payload title = %v, want %v", got, "Test Title")
	}

	if got := payload["message"]; got != "Fallback Message" {
		t.Fatalf("payload message = %v, want %v", got, "Fallback Message")
	}

	if got := payload["timestamp"]; got != event.Timestamp.Format(time.RFC3339) {
		t.Fatalf("payload timestamp = %v, want %v", got, event.Timestamp.Format(time.RFC3339))
	}
}

func TestWebhookProviderSendCustomFields(t *testing.T) {
	var receivedHeaders http.Header
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewWebhookProvider()
	event := &notification.Event{
		Type:      "system.test",
		Level:     notification.LevelInfo,
		Title:     "A title",
		Message:   "A message",
		Timestamp: time.Unix(1700000000, 0),
		Data: map[string]interface{}{
			"rendered_content": "Rendered body",
		},
	}

	err := provider.Send(context.Background(), map[string]interface{}{
		"url":    server.URL,
		"method": "POST",
		"headers": map[string]interface{}{
			"apiKey":       "secret-key",
			"Content-Type": "application/json",
		},
		"body_fields": map[string]interface{}{
			"title":   "title",
			"content": "content",
			"msgType": "text",
		},
	}, event)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if got := receivedHeaders.Get("apiKey"); got != "secret-key" {
		t.Fatalf("apiKey header = %q, want %q", got, "secret-key")
	}

	if got := receivedBody["title"]; got != "A title" {
		t.Fatalf("payload title = %v, want %v", got, "A title")
	}

	if got := receivedBody["content"]; got != "Rendered body" {
		t.Fatalf("payload content = %v, want %v", got, "Rendered body")
	}

	if got := receivedBody["msgType"]; got != "text" {
		t.Fatalf("payload msgType = %v, want %v", got, "text")
	}
}
