package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/model"
)

type stubProvider struct {
	mu     sync.Mutex
	called bool
	err    error
}

func (p *stubProvider) Type() string { return model.ChannelTypeWebhook }
func (p *stubProvider) Validate(_ map[string]interface{}) error { return nil }
func (p *stubProvider) Send(_ context.Context, _ map[string]interface{}, _ *Event) error {
	p.mu.Lock()
	p.called = true
	p.mu.Unlock()
	return p.err
}
func (p *stubProvider) WasCalled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.called
}

func TestManager_processJob_Success(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	now := time.Now()

	// claim queued -> processing
	mock.ExpectExec("UPDATE \\\"notification_jobs\\\"").WillReturnResult(sqlmock.NewResult(0, 1))

	// load job
	mock.ExpectQuery("SELECT (.+) FROM \\\"notification_jobs\\\"").WillReturnRows(
		sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"log_id",
			"channel_id",
			"provider_type",
			"event_type",
			"event_level",
			"event_title",
			"event_message",
			"event_data",
			"template_name",
			"status",
			"retry_count",
			"next_run_at",
			"last_error",
			"last_attempt_at",
		}).AddRow(
			10,
			now,
			now,
			1,
			1,
			model.ChannelTypeWebhook,
			"system.test",
			"info",
			"Title",
			"Message",
			[]byte(`{"foo":"bar"}`),
			"default_markdown",
			model.NotificationJobStatusQueued,
			0,
			now,
			"",
			nil,
		),
	)

	// load channel
	mock.ExpectQuery("SELECT (.+) FROM \\\"notification_channels\\\"").WillReturnRows(
		sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"name",
			"type",
			"enabled",
			"description",
			"config",
			"events",
		}).AddRow(
			1,
			now,
			now,
			"test",
			model.ChannelTypeWebhook,
			true,
			"",
			[]byte(`{}`),
			"{\"*\"}",
		),
	)

	// log status: pending -> sending
	mock.ExpectExec("UPDATE \\\"notification_logs\\\" SET \\\"status\\\"").
		WithArgs("sending", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// log status: sending -> success
	mock.ExpectExec("UPDATE \\\"notification_logs\\\"").WillReturnResult(sqlmock.NewResult(0, 1))

	// job status: processing -> succeeded
	mock.ExpectExec("UPDATE \\\"notification_jobs\\\"").WillReturnResult(sqlmock.NewResult(0, 1))

	prov := &stubProvider{}

	m := &Manager{
		db:        gdb,
		logger:    logx.WithContext(context.Background()),
		providers: map[string]NotificationProvider{model.ChannelTypeWebhook: prov},
	}

	m.processJob(context.Background(), 10)

	if !prov.WasCalled() {
		t.Fatalf("expected provider.Send to be called")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
