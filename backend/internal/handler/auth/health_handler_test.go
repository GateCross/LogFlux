package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"logflux/internal/middleware"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var errPingBroken = errors.New("数据库连接已断开")

// 数据库 ping 失败时健康检查须返回 503，且状态码不能被响应中间件改写成 200。
func TestHealthHandlerReportsUnavailableWhenDatabaseBroken(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("构造 sqlmock 失败: %v", err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("构造 gorm 实例失败: %v", err)
	}

	mock.ExpectPing().WillReturnError(errPingBroken)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	middleware.ResponseMiddleware(HealthHandler(db))(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("数据库异常时应返回 503，实际 %d，响应体 %s", rec.Code, rec.Body.String())
	}
}

// 数据库 ping 成功时健康检查返回 200。
func TestHealthHandlerReportsOKWhenDatabaseReachable(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("构造 sqlmock 失败: %v", err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("构造 gorm 实例失败: %v", err)
	}

	mock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	middleware.ResponseMiddleware(HealthHandler(db))(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("数据库正常时应返回 200，实际 %d，响应体 %s", rec.Code, rec.Body.String())
	}
}
