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
	"logflux/model"
)

func TestGetNotificationLogs_MapsSendingStatusTo1(t *testing.T) {
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

	// list
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
	}).AddRow(
		uint(1),
		now,
		nil,
		nil,
		"system.test",
		[]byte(`{"title":"t","message":"m","level":"info"}`),
		"sending",
		"",
		nil,
		false,
		nil,
	)
	mock.ExpectQuery("SELECT (.+) FROM \\\"notification_logs\\\"").WillReturnRows(rows)

	logic := NewGetNotificationLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetNotificationLogs(&types.LogListReq{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("GetNotificationLogs() error = %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.List))
	}
	if resp.List[0].Status != 1 {
		t.Fatalf("expected status=1 for sending, got %d", resp.List[0].Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestGetNotificationLogs_FilterBySending(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	status := 1

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM \\\"notification_logs\\\" LEFT JOIN notification_jobs").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

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
	}).AddRow(
		uint(1),
		now,
		nil,
		nil,
		"system.test",
		[]byte(`{"title":"t","message":"m","level":"info"}`),
		"sending",
		"",
		nil,
		false,
		nil,
	)
	mock.ExpectQuery("SELECT (.+) FROM \\\"notification_logs\\\" LEFT JOIN notification_jobs").
		WillReturnRows(rows)

	logic := NewGetNotificationLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetNotificationLogs(&types.LogListReq{Page: 1, PageSize: 20, Status: &status})
	if err != nil {
		t.Fatalf("GetNotificationLogs() error = %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.List))
	}
	if resp.List[0].Status != 1 {
		t.Fatalf("expected status=1 for sending, got %d", resp.List[0].Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

var _ = model.NotificationStatusPending
