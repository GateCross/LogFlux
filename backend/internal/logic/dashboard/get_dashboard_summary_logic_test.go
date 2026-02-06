package dashboard

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

func TestGetDashboardSummary_StatusCountsUseIndependentFilters(t *testing.T) {
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "caddy_logs" WHERE \(?log_time >= \$1 AND log_time <= \$2\)?$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "caddy_logs" WHERE \(?log_time >= \$1 AND log_time <= \$2\)? AND \(?status IN \(\$3,\$4\)\)?$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "caddy_logs" WHERE \(?log_time >= \$1 AND log_time <= \$2\)? AND \(?status >= \$3 AND status < \$4\)?$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "caddy_logs" WHERE \(?log_time >= \$1 AND log_time <= \$2\)? AND \(?status >= \$3 AND status < \$4\)?$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT COUNT\(DISTINCT COALESCE\(NULLIF\(client_ip, ''\), remote_ip\)\)`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

	mock.ExpectQuery(`SELECT COUNT\(DISTINCT remote_ip\) FROM caddy_logs WHERE log_time BETWEEN \$1 AND \$2 AND remote_ip <> ''`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(6))

	mock.ExpectQuery(`SELECT COUNT\(DISTINCT remote_ip\) FROM caddy_logs WHERE log_time BETWEEN \$1 AND \$2 AND status IN \(\$3,\$4\) AND remote_ip <> ''`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	mock.ExpectQuery(`SELECT floor\(extract\(epoch from log_time\) / \$1\) \* \$2 AS bucket, COUNT\(\*\) AS count`).
		WillReturnRows(sqlmock.NewRows([]string{"bucket", "count"}))

	mock.ExpectQuery(`SELECT COALESCE\(NULLIF\(country, ''\), '未知'\) AS name, COUNT\(\*\) AS value`).
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))

	mock.ExpectQuery(`SELECT \* FROM "caddy_logs" WHERE \(?log_time >= \$1 AND log_time <= \$2\)? ORDER BY log_time desc, ?id desc LIMIT \$3`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "log_time", "country", "province", "city",
			"host", "method", "uri", "proto", "status", "size", "user_agent", "remote_ip", "client_ip", "raw_log", "extra_data",
		}))

	start := time.Date(2026, 2, 6, 8, 0, 0, 0, time.Local)
	end := start.Add(time.Hour)

	logic := NewGetDashboardSummaryLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetDashboardSummary(&types.DashboardSummaryReq{
		StartTime:   start.Format("2006-01-02 15:04:05"),
		EndTime:     end.Format("2006-01-02 15:04:05"),
		IntervalSec: 60,
		TopN:        6,
		RecentLimit: 6,
	})
	if err != nil {
		t.Fatalf("GetDashboardSummary() error = %v", err)
	}
	if resp.ErrorStats.Blocked4xx != 2 {
		t.Fatalf("expected blocked4xx=2, got %d", resp.ErrorStats.Blocked4xx)
	}
	if resp.ErrorStats.Error4xx != 3 {
		t.Fatalf("expected error4xx=3, got %d", resp.ErrorStats.Error4xx)
	}
	if resp.ErrorStats.Error5xx != 1 {
		t.Fatalf("expected error5xx=1, got %d", resp.ErrorStats.Error5xx)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
