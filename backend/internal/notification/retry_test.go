package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/model"
)

func TestManager_processJob_Failure_RequeuesWithBackoff(t *testing.T) {
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

	// load job (retry_count=0)
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

	// load channel with retry config (maxAttempts=2)
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
			[]byte(`{"retry":{"maxAttempts":2,"baseDelay":"1s","maxDelay":"10s","factor":2,"jitter":false}}`),
			"{\"*\"}",
		),
	)

	// log status: pending -> sending
	mock.ExpectExec("UPDATE \\\"notification_logs\\\" SET \\\"status\\\"").
		WithArgs("sending", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// log status: sending -> pending (attempt failed but will retry)
	mock.ExpectExec("UPDATE \\\"notification_logs\\\" SET").
		WithArgs(sqlmock.AnyArg(), "pending", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// job should be re-queued with retry_count++ and next_run_at advanced
	mock.ExpectExec("UPDATE \\\"notification_jobs\\\" SET (.*\\\"next_run_at\\\".*\\\"retry_count\\\"|.*\\\"retry_count\\\".*\\\"next_run_at\\\")").WillReturnResult(sqlmock.NewResult(0, 1))

	prov := &stubProvider{err: errors.New("send failed")}
	m := &Manager{
		db:        gdb,
		logger:    logx.WithContext(context.Background()),
		providers: map[string]NotificationProvider{model.ChannelTypeWebhook: prov},
	}

	m.processJob(context.Background(), 10)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
