package notification

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/svc"
	"logflux/internal/types"
)

func TestGetNotificationLogs_IncludesJobFields(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	// count
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM \\\"notification_logs\\\"").WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(1),
	)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"channel_id",
		"rule_id",
		"event_type",
		"event_data",
		"status",
		"error_message",
		"sent_at",
		"is_read",
		"read_at",
		// joined job fields
		"job_status",
		"job_retry_count",
		"job_next_run_at",
		"job_last_error",
	}).AddRow(
		uint(1),
		now,
		uint(1),
		nil,
		"system.test",
		[]byte(`{"title":"t","message":"m","level":"info"}`),
		"pending",
		"",
		nil,
		false,
		nil,
		"queued",
		2,
		now,
		"timeout",
	)

	mock.ExpectQuery("SELECT (.+) FROM \\\"notification_logs\\\"").WillReturnRows(rows)

	logic := NewGetNotificationLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetNotificationLogs(&types.LogListReq{Page: 1, PageSize: 20, Status: -1})
	if err != nil {
		t.Fatalf("GetNotificationLogs() error=%v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.List))
	}
	if resp.List[0].JobStatus != "queued" {
		t.Fatalf("expected jobStatus=queued, got %q", resp.List[0].JobStatus)
	}
	if resp.List[0].JobRetryCount != 2 {
		t.Fatalf("expected jobRetryCount=2, got %d", resp.List[0].JobRetryCount)
	}
	if resp.List[0].LastError != "timeout" {
		t.Fatalf("expected lastError=timeout, got %q", resp.List[0].LastError)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestGetNotificationLogs_FilterByJobStatus(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM \\\"notification_logs\\\" LEFT JOIN notification_jobs").WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(0),
	)
	mock.ExpectQuery("SELECT (.+) FROM \\\"notification_logs\\\" LEFT JOIN notification_jobs").WillReturnRows(
		sqlmock.NewRows([]string{"id","created_at","channel_id","rule_id","event_type","event_data","status","error_message","sent_at","is_read","read_at","job_status","job_retry_count","job_next_run_at","job_last_error"}),
	)

	logic := NewGetNotificationLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetNotificationLogs(&types.LogListReq{Page: 1, PageSize: 20, JobStatus: "queued", Status: -1})
	if err != nil {
		t.Fatalf("GetNotificationLogs() error=%v", err)
	}
	_ = resp

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
