package caddy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/svc"
	"logflux/internal/types"
)

const testCaddyConfigWithCoraza = ":443 {\n  route {\n    coraza_waf {\n      directives `\n        SecRuleEngine Off\n      `\n    }\n  }\n}\n"

func TestPreviewWafPolicySuccess(t *testing.T) {
	gdb, mock, cleanup := newPolicyFlowMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "waf_policies"`).WillReturnRows(policyRows(now))
	mock.ExpectQuery(`SELECT .* FROM "waf_rule_exclusions"`).WillReturnRows(policyExclusionRows())

	logic := NewPreviewWafPolicyLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.PreviewWafPolicy(&types.WafPolicyActionReq{ID: 1})
	if err != nil {
		t.Fatalf("PreviewWafPolicy() error = %v", err)
	}
	if resp == nil || strings.TrimSpace(resp.Directives) == "" {
		t.Fatalf("expected non-empty directives")
	}
	if !strings.Contains(resp.Directives, "SecRuleEngine On") {
		t.Fatalf("unexpected directives content: %s", resp.Directives)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestValidateWafPolicySuccess(t *testing.T) {
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adapt" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newPolicyFlowMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "waf_policies"`).WillReturnRows(policyRows(now))
	mock.ExpectQuery(`SELECT .* FROM "waf_rule_exclusions"`).WillReturnRows(policyExclusionRows())
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(caddyServerRows(now, caddyMock.URL, testCaddyConfigWithCoraza))

	logic := NewValidateWafPolicyLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.ValidateWafPolicy(&types.WafPolicyActionReq{ID: 1})
	if err != nil {
		t.Fatalf("ValidateWafPolicy() error = %v", err)
	}
	if resp == nil || resp.Code != 200 {
		t.Fatalf("expected code=200, got %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestPublishWafPolicyValidateFailedLocalized(t *testing.T) {
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adapt" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if !strings.Contains(string(bodyBytes), "SecRuleEngine On") {
			http.Error(w, "invalid candidate config", http.StatusBadRequest)
			return
		}
		http.Error(w, "adapt failed for publish", http.StatusInternalServerError)
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newPolicyFlowMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "waf_policies"`).WillReturnRows(policyRows(now))
	mock.ExpectQuery(`SELECT .* FROM "waf_policy_bindings"`).WillReturnRows(policyBindingConflictRows())
	mock.ExpectQuery(`SELECT .* FROM "waf_rule_exclusions"`).WillReturnRows(policyExclusionRows())
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(caddyServerRows(now, caddyMock.URL, testCaddyConfigWithCoraza))

	logic := NewPublishWafPolicyLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	_, err := logic.PublishWafPolicy(&types.WafPolicyActionReq{ID: 1})
	if err == nil {
		t.Fatalf("expected publish validate error")
	}
	if !strings.Contains(err.Error(), "策略发布前校验失败") {
		t.Fatalf("expected localized publish validate error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestRollbackWafPolicyValidateFailedLocalized(t *testing.T) {
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adapt" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "adapt failed for rollback", http.StatusInternalServerError)
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newPolicyFlowMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "waf_policy_revisions"`).WillReturnRows(policyRevisionRows(now))
	mock.ExpectQuery(`SELECT .* FROM "waf_policies"`).WillReturnRows(policyRows(now))
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(caddyServerRows(now, caddyMock.URL, testCaddyConfigWithCoraza))

	logic := NewRollbackWafPolicyLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	_, err := logic.RollbackWafPolicy(&types.WafPolicyRollbackReq{RevisionId: 10})
	if err == nil {
		t.Fatalf("expected rollback validate error")
	}
	if !strings.Contains(err.Error(), "策略回滚前校验失败") {
		t.Fatalf("expected localized rollback validate error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func newPolicyFlowMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
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

func policyRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
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
	)
}

func policyRevisionRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at",
		"policy_id", "version", "status",
		"config_snapshot", "directives_snapshot",
		"operator", "message",
	}).AddRow(
		uint(10), now, now,
		uint(1), uint(2), "published",
		[]byte(`{}`),
		"SecRuleEngine DetectionOnly\nSecAuditEngine RelevantOnly\nSecAuditLogFormat JSON\nSecAuditLogRelevantStatus ^(?:5|4(?!04))\nSecRequestBodyAccess On\nSecRequestBodyLimit 10485760\nSecRequestBodyNoFilesLimit 1048576\nSecAction \"id:900000,phase:1,pass,nolog,t:none,setvar:tx.paranoia_level=2\"\nSecAction \"id:900110,phase:1,pass,nolog,t:none,setvar:tx.inbound_anomaly_score_threshold=5\"\nSecAction \"id:900100,phase:1,pass,nolog,t:none,setvar:tx.outbound_anomaly_score_threshold=4\"",
		"system", "publish policy",
	)
}

func caddyServerRows(now time.Time, url, config string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at",
		"name", "url", "token", "type", "username", "password",
		"config", "modules",
	}).AddRow(
		uint(1), now, now,
		"local-default", url, "", "local", "", "",
		config, "{}",
	)
}

func policyExclusionRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at",
		"policy_id", "name", "description", "enabled",
		"scope_type", "host", "path", "method",
		"remove_type", "remove_value",
	})
}

func policyBindingConflictRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"scope_type", "host", "path", "method", "priority", "count",
	})
}
