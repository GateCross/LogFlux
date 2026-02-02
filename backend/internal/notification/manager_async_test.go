package notification

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/notification/template"
	"logflux/model"
)

type recordingProvider struct {
	mu     sync.Mutex
	called bool
	err    error
}

func (p *recordingProvider) Type() string { return model.ChannelTypeWebhook }

func (p *recordingProvider) Validate(_ map[string]interface{}) error { return nil }

func (p *recordingProvider) Send(_ context.Context, _ map[string]interface{}, _ *Event) error {
	p.mu.Lock()
	p.called = true
	p.mu.Unlock()
	return p.err
}

func (p *recordingProvider) WasCalled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.called
}

func TestManager_Notify_EnqueuesJobsAndReturns(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	// DefaultTemplateManager 不依赖 DB；并且测试不需要 gorm 开启事务
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	// Expect: create notification_logs row
	mock.ExpectQuery("INSERT INTO \\\"notification_logs\\\"").
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(1),
		)

	// Expect: create notification_jobs row
	mock.ExpectQuery("INSERT INTO \\\"notification_jobs\\\"").
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(10),
		)


	prov := &recordingProvider{err: errors.New("should not be called")}

	tm := template.NewTemplateManager(gdb)
	// 避免 ensureDefaults 写库（异步 goroutine），这里不调用 LoadTemplates；直接用默认模板即可

	m := &Manager{
		db:          gdb,
		logger:      logx.WithContext(context.Background()),
		started:     true,
		providers:   map[string]NotificationProvider{model.ChannelTypeWebhook: prov},
		templateMgr: tm,
		channels: map[uint]*model.NotificationChannel{
			1: {
				ID:      1,
				Name:    "test",
				Type:    model.ChannelTypeWebhook,
				Enabled: true,
				Events:  model.StringArray{"*"},
				Config:  model.JSONMap{"url": "http://example.invalid"},
			},
		},
		rules:      map[uint]*model.NotificationRule{},
		ruleEngine: NewRuleEngine(nil),
	}

	event := NewEvent("system.test", LevelInfo, "Title", "Message")

	start := time.Now()
	if err := m.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	_ = start

	if prov.WasCalled() {
		t.Fatalf("expected provider.Send not to be called during Notify")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
