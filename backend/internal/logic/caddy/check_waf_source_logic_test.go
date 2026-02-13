package caddy

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

func TestCheckWafSourceRejectsNonHTTPSAndUpdatesAudit(t *testing.T) {
	db, mock, cleanup := newCheckSourceMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "waf_sources"`).WillReturnRows(
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
			"http://example.com/crs.tar.gz", "", "",
			"none", "",
			"",
			true, true, true, false,
			nil, "", "",
			[]byte(`{}`),
		),
	)
	mock.ExpectQuery(`INSERT INTO "waf_update_jobs"`).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(uint(100)),
	)
	mock.ExpectExec(`UPDATE "waf_sources" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "waf_update_jobs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))

	logic := NewCheckWafSourceLogic(context.Background(), &svc.ServiceContext{DB: db})
	_, err := logic.CheckWafSource(&types.WafSourceActionReq{ID: 1})
	if err == nil || !strings.Contains(err.Error(), "only https url is allowed") {
		t.Fatalf("expected only https url error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestCheckWafSourceSuccess(t *testing.T) {
	db, mock, cleanup := newCheckSourceMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "waf_sources"`).WillReturnRows(
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
			"https://example.com/crs.tar.gz", "", "http://127.0.0.1:8080",
			"none", "",
			"",
			true, true, true, false,
			nil, "", "",
			[]byte(`{}`),
		),
	)
	mock.ExpectQuery(`INSERT INTO "waf_update_jobs"`).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(uint(101)),
	)
	mock.ExpectExec(`UPDATE "waf_sources" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "waf_update_jobs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))

	logic := NewCheckWafSourceLogic(context.Background(), &svc.ServiceContext{DB: db})
	resp, err := logic.CheckWafSource(&types.WafSourceActionReq{ID: 1})
	if err != nil {
		t.Fatalf("CheckWafSource() error = %v", err)
	}
	if resp == nil || resp.Code != 200 {
		t.Fatalf("expected success response, got %+v", resp)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func newCheckSourceMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		_ = sqldb.Close()
		t.Fatalf("failed to open gorm: %v", err)
	}

	cleanup := func() {
		_ = sqldb.Close()
	}
	return gdb, mock, cleanup
}
