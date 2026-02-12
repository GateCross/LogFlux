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

func TestListWafReleasesSuccess(t *testing.T) {
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
	mock.ExpectQuery(`SELECT count\(\*\) FROM "waf_releases"`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(1),
	)
	mock.ExpectQuery(`SELECT \* FROM "waf_releases"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"source_id", "kind", "version", "artifact_type",
			"checksum", "size_bytes", "storage_path", "status", "meta",
		}).AddRow(
			uint(1), now, now,
			uint(2), "crs", "v4.23.0", "tar.gz",
			"sha256:demo", int64(1024), "/config/security/releases/v4.23.0", "active", []byte(`{}`),
		),
	)

	logic := NewListWafReleasesLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.ListWafReleases(&types.WafReleaseListReq{
		Page:     1,
		PageSize: 20,
		Kind:     "crs",
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("ListWafReleases() error = %v", err)
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
	if resp.List[0].Version != "v4.23.0" {
		t.Fatalf("expected version=v4.23.0, got %s", resp.List[0].Version)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestListWafReleasesCorazaFilterReturnsEmpty(t *testing.T) {
	sqldb, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	logic := NewListWafReleasesLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.ListWafReleases(&types.WafReleaseListReq{
		Page:     1,
		PageSize: 20,
		Kind:     "coraza_engine",
	})
	if err != nil {
		t.Fatalf("ListWafReleases() error = %v", err)
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
}
