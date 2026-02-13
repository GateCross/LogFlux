package caddy

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/config"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func TestSyncWafSourceGuardFailures(t *testing.T) {
	cases := []struct {
		name       string
		sourceKind string
		sourceMode string
		enabled    bool
		url        string
		wantErr    string
	}{
		{
			name:       "disabled source",
			sourceKind: "crs",
			sourceMode: "remote",
			enabled:    false,
			url:        "https://example.com/crs.tar.gz",
			wantErr:    "source is disabled",
		},
		{
			name:       "coraza engine source",
			sourceKind: "coraza_engine",
			sourceMode: "remote",
			enabled:    true,
			url:        "https://example.com/coraza.tar.gz",
			wantErr:    "Coraza 引擎更新源无需手工同步",
		},
		{
			name:       "non-remote source mode",
			sourceKind: "crs",
			sourceMode: "manual",
			enabled:    true,
			url:        "https://example.com/crs.tar.gz",
			wantErr:    "source mode is not remote",
		},
		{
			name:       "empty source url",
			sourceKind: "crs",
			sourceMode: "remote",
			enabled:    true,
			url:        "   ",
			wantErr:    "source url is empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, cleanup := newSyncSourceMockDB(t)
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
					"default-source", tc.sourceKind, tc.sourceMode,
					tc.url, "", "",
					"none", "",
					"",
					tc.enabled, true, true, false,
					nil, "", "",
					[]byte(`{}`),
				),
			)

			logic := NewSyncWafSourceLogic(context.Background(), &svc.ServiceContext{
				DB: db,
				Config: config.Config{
					Waf: config.WafConf{
						WorkDir: t.TempDir(),
					},
				},
			})

			_, err := logic.SyncWafSource(&types.WafSourceSyncReq{ID: 1})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sql expectations not met: %v", err)
			}
		})
	}
}

func TestNormalizeWafSyncFetchError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		hasProxy bool
		want     string
	}{
		{
			name:     "timeout without proxy",
			err:      fmt.Errorf("context deadline exceeded"),
			hasProxy: false,
			want:     "下载源超时，请配置可用代理后重试",
		},
		{
			name:     "timeout with proxy",
			err:      fmt.Errorf("i/o timeout"),
			hasProxy: true,
			want:     "下载源超时（代理与直连均失败），请检查代理连通性或稍后重试",
		},
		{
			name:     "host not allowed",
			err:      fmt.Errorf("host not allowed: github.com"),
			hasProxy: false,
			want:     "下载源域名未加入允许列表，请联系管理员在 Waf.AllowedDomains 中添加该域名",
		},
		{
			name:     "passthrough unknown error",
			err:      fmt.Errorf("unexpected status code: 503"),
			hasProxy: false,
			want:     "unexpected status code: 503",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeWafSyncFetchError(tc.err, tc.hasProxy)
			if got == nil {
				t.Fatalf("expected non-nil error")
			}
			if got.Error() != tc.want {
				t.Fatalf("unexpected error, want=%q got=%q", tc.want, got.Error())
			}
		})
	}
}

func newSyncSourceMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
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
