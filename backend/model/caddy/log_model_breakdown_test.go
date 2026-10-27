package caddy

import (
	"context"
	"database/sql/driver"
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

// 访问日志 / 错误日志只按 logger 命名空间分流，两侧必须是互补条件：
// 用等值比较会漏掉 Caddy 自动生成的 http.log.access.log0 等子名，
// 把 level IN (...) 混进种类判定则会让 5xx 的访问日志同时出现在两个视图。
func TestList_LogKindPartitionsByLoggerNamespace(t *testing.T) {
	cases := []struct {
		name string
		kind string
		// 断言生成的 WHERE 子句与绑定参数
		wantSQL  string
		wantArgs []driver.Value
	}{
		{
			name:     "访问日志：空值兜底 + 命名空间前缀",
			kind:     CaddyLogKindAccess,
			wantSQL:  `WHERE COALESCE(logger, '') = '' OR logger = $1 OR logger LIKE $2`,
			wantArgs: []driver.Value{"http.log.access", "http.log.access.%"},
		},
		{
			name:     "错误日志：访问命名空间之外",
			kind:     CaddyLogKindError,
			wantSQL:  `WHERE COALESCE(logger, '') <> '' AND logger <> $1 AND logger NOT LIKE $2`,
			wantArgs: []driver.Value{"http.log.access", "http.log.access.%"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sqldb, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("open sqlmock: %v", err)
			}
			defer sqldb.Close()

			db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open gorm: %v", err)
			}

			mock.ExpectQuery(regexp.QuoteMeta(c.wantSQL)).
				WithArgs(c.wantArgs...).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "caddy_logs"`)).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			if _, _, err := NewCaddyLogModel(db).List(context.Background(), CaddyLogQuery{
				Status:   -1,
				Page:     1,
				PageSize: 20,
				LogKind:  c.kind,
			}); err != nil {
				t.Fatalf("List error: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("WHERE 子句未按命名空间互补生成: %v", err)
			}
		})
	}
}

// 写入路径漏填 level / logger 时由 BeforeCreate 兜底，
// 否则该行在两个视图里都查不到（历史事故的成因）。
func TestCaddyLog_BeforeCreateFillsLevelAndLogger(t *testing.T) {
	cases := []struct {
		name       string
		level      string
		logger     string
		wantLevel  string
		wantLogger string
	}{
		{name: "两个都空", wantLevel: "info", wantLogger: "http.log.access"},
		{name: "只有 level 空", level: "", logger: "tls", wantLevel: "info", wantLogger: "tls"},
		{name: "只有 logger 空", level: "error", wantLevel: "error", wantLogger: "http.log.access"},
		{name: "已有值不覆盖", level: "warn", logger: "http.log.error", wantLevel: "warn", wantLogger: "http.log.error"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			entry := &CaddyLog{Level: c.level, Logger: c.logger}
			if err := entry.BeforeCreate(nil); err != nil {
				t.Fatalf("BeforeCreate error: %v", err)
			}
			if entry.Level != c.wantLevel || entry.Logger != c.wantLogger {
				t.Fatalf("got level=%q logger=%q, want level=%q logger=%q",
					entry.Level, entry.Logger, c.wantLevel, c.wantLogger)
			}
		})
	}
}

// hook 必须真的挂在 Create 上：只测 BeforeCreate 本身无法发现「字段没加 hook」这类问题。
func TestCaddyLogCreateAppliesHook(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sqlmock: %v", err)
	}
	defer sqldb.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}

	// 19 个占位符，顺序同 INSERT 列：level 在第 7、logger 在第 8，此处断言已兜底成 info / http.log.access
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "caddy_logs"`)).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			"info", caddyAccessLoggerName,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	err = NewCaddyLogModel(db).Create(context.Background(), &CaddyLog{
		Host:      "a.example.com",
		RawLog:    "{}",
		ExtraData: "{}",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("INSERT 未带上兜底后的 level/logger: %v", err)
	}
}
