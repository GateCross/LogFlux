package caddy

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

func TestListWafJobsSuccess(t *testing.T) {
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	now := time.Now()
	startedAt := now.Add(-2 * time.Minute)
	finishedAt := now.Add(-1 * time.Minute)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "waf_update_jobs"`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(1),
	)
	mock.ExpectQuery(`SELECT \* FROM "waf_update_jobs"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"source_id", "release_id",
			"action", "trigger_mode", "operator",
			"status", "message",
			"started_at", "finished_at", "meta",
		}).AddRow(
			uint(1), now, now,
			uint(2), uint(3),
			"download", "manual", "1",
			"success", "sync success",
			&startedAt, &finishedAt, []byte(`{}`),
		),
	)
	mock.ExpectQuery(`SELECT .* FROM "users" WHERE id IN`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "username"}).AddRow(uint(1), "admin"),
	)

	logic := NewListWafJobsLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.ListWafJobs(&types.WafJobListReq{
		Page:     1,
		PageSize: 20,
		Status:   "success",
		Action:   "download",
	})
	if err != nil {
		t.Fatalf("ListWafJobs() error = %v", err)
	}
	if resp == nil {
		t.Fatalf("expected non-nil response")
	}
	if resp.Total != 1 {
		t.Fatalf("expected total=1, got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected list size=1, got %d", len(resp.List))
	}
	if resp.List[0].Operator != "admin" {
		t.Fatalf("expected operator to map to username admin, got %s", resp.List[0].Operator)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
