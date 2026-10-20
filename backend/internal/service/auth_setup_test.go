package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	usermodel "logflux/model/user"

	"logflux/internal/svc"
	"logflux/internal/types"
)

// newSetupTestService 构造一个只依赖 users 表的 AuthService，用于验证初始化逻辑。
func newSetupTestService(t *testing.T) (*AuthService, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("打开 sqlmock 失败: %v", err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })

	gdb, err := gorm.Open(
		postgres.New(postgres.Config{Conn: sqldb}),
		&gorm.Config{SkipDefaultTransaction: true},
	)
	if err != nil {
		t.Fatalf("打开 gorm 失败: %v", err)
	}

	svcCtx := &svc.ServiceContext{
		DB:        gdb,
		UserModel: usermodel.NewUserModel(gdb),
	}
	return NewAuthService(context.Background(), svcCtx), mock
}

func TestGetSetupStatus(t *testing.T) {
	cases := []struct {
		name        string
		count       int64
		initialized bool
	}{
		{"空库未初始化", 0, false},
		{"已有账号已初始化", 2, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authSvc, mock := newSetupTestService(t)
			mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.count))

			resp, err := authSvc.GetSetupStatus()
			if err != nil {
				t.Fatalf("GetSetupStatus 返回错误: %v", err)
			}
			if resp.Initialized != tc.initialized {
				t.Fatalf("Initialized = %v, want %v", resp.Initialized, tc.initialized)
			}
		})
	}
}

func TestSetupRejectsWeakPassword(t *testing.T) {
	authSvc, _ := newSetupTestService(t)

	// 密码校验在查库之前完成，因此这里不需要 mock 任何 SQL
	_, err := authSvc.Setup(&types.SetupReq{Username: "admin", Password: "abc"})
	if err == nil {
		t.Fatal("弱密码应被拒绝，但 Setup 返回成功")
	}
}

func TestSetupRejectsEmptyUsername(t *testing.T) {
	authSvc, _ := newSetupTestService(t)

	_, err := authSvc.Setup(&types.SetupReq{Username: "   ", Password: "abcd1234"})
	if err == nil {
		t.Fatal("空用户名应被拒绝，但 Setup 返回成功")
	}
}

func TestSetupRejectsWhenAlreadyInitialized(t *testing.T) {
	authSvc, mock := newSetupTestService(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	_, err := authSvc.Setup(&types.SetupReq{Username: "admin", Password: "abcd1234"})
	if err == nil {
		t.Fatal("系统已初始化时 Setup 应被拒绝")
	}
}
