package log

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/svc"
	"logflux/internal/types"
)

func TestGetSystemLogs_FilterSortAndPagination(t *testing.T) {
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "system_logs"`).WillReturnRows(countRows)

	logTime := time.Date(2026, 2, 9, 12, 0, 0, 0, time.Local)
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "log_time", "level", "message", "caller", "trace_id", "span_id", "source", "file_path", "raw_log", "extra_data",
	}).AddRow(
		uint(101),
		time.Now(),
		time.Now(),
		logTime,
		"error",
		"runtime panic",
		"internal/logic/demo.go:10",
		"trace-1",
		"span-1",
		"backend",
		"/var/log/logflux/backend.log",
		`{"raw":"panic"}`,
		`{"module":"runtime"}`,
	)
	mock.ExpectQuery(`SELECT \* FROM "system_logs"`).WillReturnRows(rows)

	logic := NewGetSystemLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetSystemLogs(&types.SystemLogReq{
		Page:      2,
		PageSize:  10,
		Keyword:   "panic",
		Source:    "backend",
		Level:     "ERROR",
		StartTime: "2026-02-09 00:00:00",
		EndTime:   "2026-02-10 00:00:00",
		SortBy:    "logTime",
		Order:     "asc",
	})
	if err != nil {
		t.Fatalf("GetSystemLogs() error = %v", err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected total=1, got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.List))
	}

	item := resp.List[0]
	if item.ID != 101 {
		t.Fatalf("expected id=101, got %d", item.ID)
	}
	if item.Level != "error" {
		t.Fatalf("expected level=error, got %s", item.Level)
	}
	if item.Source != "backend" {
		t.Fatalf("expected source=backend, got %s", item.Source)
	}
	if item.LogTime != "2026-02-09 12:00:00" {
		t.Fatalf("expected formatted logTime, got %s", item.LogTime)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestGetSystemLogs_InvalidTimeRange(t *testing.T) {
	sqldb, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	logic := NewGetSystemLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})

	_, err = logic.GetSystemLogs(&types.SystemLogReq{
		Page:      1,
		PageSize:  20,
		StartTime: "invalid-time",
	})
	if err == nil {
		t.Fatalf("expected invalid time parse error")
	}

	if !strings.Contains(err.Error(), "invalid startTime") {
		t.Fatalf("expected invalid startTime error, got %v", err)
	}
}

func TestGetSystemLogs_EmptyResult(t *testing.T) {
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "system_logs"`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(0),
	)
	mock.ExpectQuery(`SELECT \* FROM "system_logs"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_at", "updated_at", "log_time", "level", "message", "caller", "trace_id", "span_id", "source", "file_path", "raw_log", "extra_data"}),
	)

	logic := NewGetSystemLogsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetSystemLogs(&types.SystemLogReq{
		Page:     1,
		PageSize: 20,
		SortBy:   "logTime",
		Order:    "desc",
	})
	if err != nil {
		t.Fatalf("GetSystemLogs() error = %v", err)
	}

	if resp.Total != 0 {
		t.Fatalf("expected total=0, got %d", resp.Total)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list, got %d", len(resp.List))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
