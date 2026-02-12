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

func TestListWafSourcesSuccess(t *testing.T) {
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
	mock.ExpectQuery(`SELECT count\(\*\) FROM "waf_sources"`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(1),
	)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "waf_sources"`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(1),
	)
	mock.ExpectQuery(`SELECT \* FROM "waf_sources"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"name", "kind", "mode",
			"url", "checksum_url", "proxy_url",
			"auth_type", "auth_secret",
			"schedule",
			"enabled", "auto_check", "auto_download", "auto_activate",
			"last_checked_at", "last_release", "last_error",
			"meta",
		}).AddRow(
			uint(1), now, now,
			"default-crs", "crs", "remote",
			"https://example.com/crs.tar.gz", "", "",
			"none", "",
			"0 0 */6 * * *",
			true, true, true, false,
			nil, "v4.23.0", "",
			[]byte(`{"default":true}`),
		),
	)

	logic := NewListWafSourcesLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.ListWafSources(&types.WafSourceListReq{
		Page:     1,
		PageSize: 20,
		Kind:     "crs",
	})
	if err != nil {
		t.Fatalf("ListWafSources() error = %v", err)
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
	if resp.List[0].Name != "default-crs" {
		t.Fatalf("expected source name=default-crs, got %s", resp.List[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestListWafSourcesCorazaFilterReturnsEmpty(t *testing.T) {
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "waf_sources"`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(1),
	)

	logic := NewListWafSourcesLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.ListWafSources(&types.WafSourceListReq{
		Page:     1,
		PageSize: 20,
		Kind:     "coraza_engine",
	})
	if err != nil {
		t.Fatalf("ListWafSources() error = %v", err)
	}
	if resp == nil {
		t.Fatalf("expected non-nil response")
	}
	if resp.Total != 0 {
		t.Fatalf("expected total=0, got %d", resp.Total)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
