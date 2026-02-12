package caddy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/svc"
	"logflux/internal/types"
)

func TestPublishWafPolicyLoadFailedRollbackToLastGood(t *testing.T) {
	lastGoodConfig := ":443 {\n  route {\n    coraza_waf {\n      directives `\n        SecRuleEngine Off\n      `\n    }\n  }\n}\n"

	type caddyCall struct {
		path string
		body string
	}
	var (
		mu        sync.Mutex
		calls     []caddyCall
		loadCount int
	)

	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		mu.Lock()
		calls = append(calls, caddyCall{path: r.URL.Path, body: string(bodyBytes)})
		mu.Unlock()

		switch r.URL.Path {
		case "/adapt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/load":
			mu.Lock()
			loadCount++
			currentLoadCount := loadCount
			mu.Unlock()

			if currentLoadCount == 1 {
				http.Error(w, "candidate load failed", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer caddyMock.Close()

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
	mock.ExpectQuery(`SELECT .* FROM "waf_policies"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"name", "description", "enabled", "is_default",
			"engine_mode", "audit_engine", "audit_log_format", "audit_relevant_status",
			"request_body_access", "request_body_limit", "request_body_no_files_limit",
			"crs_template", "crs_paranoia_level", "crs_inbound_anomaly_threshold", "crs_outbound_anomaly_threshold",
			"config",
		}).AddRow(
			uint(1), now, now,
			"default-runtime-policy", "", true, true,
			"on", "relevantonly", "json", "^(?:5|4(?!04))",
			true, int64(10485760), int64(1048576),
			"balanced", int64(2), int64(5), int64(4),
			[]byte(`{}`),
		),
	)
	mock.ExpectQuery(`SELECT .* FROM "waf_policy_bindings"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"scope_type", "host", "path", "method", "priority", "count",
		}),
	)
	mock.ExpectQuery(`SELECT .* FROM "waf_rule_exclusions"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"policy_id", "name", "description", "enabled",
			"scope_type", "host", "path", "method",
			"remove_type", "remove_value",
		}),
	)
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"name", "url", "token", "type", "username", "password",
			"config", "modules",
		}).AddRow(
			uint(1), now, now,
			"local-default", caddyMock.URL, "", "local", "", "",
			lastGoodConfig, "{}",
		),
	)

	logic := NewPublishWafPolicyLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	_, err = logic.PublishWafPolicy(&types.WafPolicyActionReq{ID: 1})
	if err == nil {
		t.Fatalf("expected publish error when first /load fails")
	}

	errMessage := err.Error()
	if !strings.Contains(errMessage, "策略发布失败") {
		t.Fatalf("expected localized publish error, got: %s", errMessage)
	}

	mu.Lock()
	defer mu.Unlock()
	expectedLastGoodConfig := strings.TrimSpace(lastGoodConfig)

	if len(calls) != 4 {
		t.Fatalf("expected 4 caddy calls (adapt/load/adapt/load), got %d", len(calls))
	}

	expectedSequence := []string{"/adapt", "/load", "/adapt", "/load"}
	for i, expectedPath := range expectedSequence {
		if calls[i].path != expectedPath {
			t.Fatalf("unexpected call[%d] path, want=%s got=%s", i, expectedPath, calls[i].path)
		}
	}

	if strings.TrimSpace(calls[0].body) == expectedLastGoodConfig || strings.TrimSpace(calls[1].body) == expectedLastGoodConfig {
		t.Fatalf("expected first adapt/load to use candidate config, got last_good")
	}
	if strings.TrimSpace(calls[2].body) != expectedLastGoodConfig {
		t.Fatalf("expected rollback adapt body to be last_good config")
	}
	if strings.TrimSpace(calls[3].body) != expectedLastGoodConfig {
		t.Fatalf("expected rollback load body to be last_good config")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
