package notification

import (
	"context"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/notification/template"
	"logflux/model"
)

type directSendProvider struct {
	mu         sync.Mutex
	lastConfig map[string]interface{}
	lastEvent  *Event
}

func (p *directSendProvider) Type() string { return model.ChannelTypeWebhook }

func (p *directSendProvider) Validate(_ map[string]interface{}) error { return nil }

func (p *directSendProvider) Send(_ context.Context, config map[string]interface{}, event *Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastConfig = config
	p.lastEvent = event
	return nil
}

func TestManagerSendToChannel(t *testing.T) {
	sqldb, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	tm := template.NewTemplateManager(gdb)
	_ = tm.LoadTemplates()

	provider := &directSendProvider{}
	m := &Manager{
		db:          gdb,
		providers:   map[string]NotificationProvider{model.ChannelTypeWebhook: provider},
		templateMgr: tm,
		channels: map[uint]*model.NotificationChannel{
			1: {
				ID:     1,
				Type:   model.ChannelTypeWebhook,
				Config: model.JSONMap{"url": "https://example.com"},
			},
		},
	}

	event := NewEvent("system.test", LevelInfo, "Test Title", "Test Content")
	if err := m.SendToChannel(context.Background(), 1, event); err != nil {
		t.Fatalf("SendToChannel() error = %v", err)
	}

	provider.mu.Lock()
	defer provider.mu.Unlock()

	if provider.lastEvent == nil {
		t.Fatal("expected provider to receive event")
	}

	rendered, ok := provider.lastEvent.Data["rendered_content"].(string)
	if !ok || rendered == "" {
		t.Fatal("expected rendered_content to be populated")
	}

	if provider.lastConfig["url"] != "https://example.com" {
		t.Fatalf("provider config url = %v", provider.lastConfig["url"])
	}
}
