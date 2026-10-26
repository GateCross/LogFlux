package caddy

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 站点指标聚合必须保持单条 SQL：总请求与 4xx/5xx 由条件聚合一次出，
// 不能退化回「一条 COUNT(*) + 两条 COUNT(*) WHERE status BETWEEN」的三次扫描。
func TestCountBreakdownByHosts_SingleQueryWithFilterAggregates(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sqlmock: %v", err)
	}
	defer sqldb.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}

	start := time.Date(2026, 3, 20, 11, 45, 0, 0, time.Local)
	end := time.Date(2026, 3, 20, 12, 0, 0, 0, time.Local)

	// 断言生成的 SQL 同时含三种计数，且状态边界为半开区间
	mock.ExpectQuery(regexp.QuoteMeta(`COUNT(*) FILTER (WHERE status >= 400 AND status < 500) AS count4xx`)).
		WillReturnRows(sqlmock.NewRows([]string{"host", "total", "count4xx", "count5xx"}).
			AddRow("a.example.com", int64(10), int64(2), int64(3)))

	rows, err := NewCaddyLogModel(db).CountBreakdownByHosts(
		context.Background(), start, end, []string{"a.example.com"},
	)
	if err != nil {
		t.Fatalf("CountBreakdownByHosts error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Host != "a.example.com" || rows[0].Total != 10 || rows[0].Count4xx != 2 || rows[0].Count5xx != 3 {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
	// 只有一条查询被消费，多扫一次都会让这里失败
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected exactly one query: %v", err)
	}
}

// 空 hosts 不发查询，直接返回空结果（与既有按 host 聚合接口契约一致）。
func TestCountBreakdownByHosts_EmptyHostsSkipsQuery(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sqlmock: %v", err)
	}
	defer sqldb.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}

	rows, err := NewCaddyLogModel(db).CountBreakdownByHosts(
		context.Background(), time.Now().Add(-time.Hour), time.Now(), nil,
	)
	if err != nil {
		t.Fatalf("CountBreakdownByHosts error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty rows, got %d", len(rows))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected no query: %v", err)
	}
}
