package notification

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/svc"
	"logflux/internal/types"
)

func TestDeleteNotificationLog_DeletesJobsAndLog(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	mock.ExpectExec("DELETE FROM \\\"notification_jobs\\\"").WithArgs(uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM \\\"notification_logs\\\"").WithArgs(uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	logic := NewDeleteNotificationLogLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	_, err = logic.DeleteNotificationLog(&types.IDReq{ID: 1})
	if err != nil {
		t.Fatalf("DeleteNotificationLog() error=%v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestBatchDeleteNotificationLogs_EmptyIDs_Noops(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	logic := NewBatchDeleteNotificationLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	_, err = logic.BatchDeleteNotificationLogs(&types.BatchDeleteNotificationLogsReq{IDs: nil})
	if err != nil {
		t.Fatalf("BatchDeleteNotificationLogs() error=%v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestBatchDeleteNotificationLogs_DeletesJobsAndLogs(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	mock.ExpectExec("DELETE FROM \\\"notification_jobs\\\"").WithArgs(uint(1), uint(2)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("DELETE FROM \\\"notification_logs\\\"").WithArgs(uint(1), uint(2)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	logic := NewBatchDeleteNotificationLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	_, err = logic.BatchDeleteNotificationLogs(&types.BatchDeleteNotificationLogsReq{IDs: []uint{1, 2}})
	if err != nil {
		t.Fatalf("BatchDeleteNotificationLogs() error=%v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestClearNotificationLogs_TruncatesTables(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	mock.ExpectExec("TRUNCATE TABLE notification_jobs RESTART IDENTITY").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("TRUNCATE TABLE notification_logs RESTART IDENTITY").
		WillReturnResult(sqlmock.NewResult(0, 0))

	logic := NewClearNotificationLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	_, err = logic.ClearNotificationLogs()
	if err != nil {
		t.Fatalf("ClearNotificationLogs() error=%v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
