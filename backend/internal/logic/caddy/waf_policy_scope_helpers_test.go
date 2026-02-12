package caddy

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/model"
)

func TestBuildWafRuleExclusionDirectives(t *testing.T) {
	exclusions := []model.WafRuleExclusion{
		{
			Enabled:     true,
			ScopeType:   "global",
			RemoveType:  "id",
			RemoveValue: "920350",
		},
		{
			Enabled:     true,
			ScopeType:   "route",
			Host:        "app.example.com",
			Path:        "/api/login",
			Method:      "POST",
			RemoveType:  "tag",
			RemoveValue: "attack-sqli",
		},
	}

	directives, err := buildWafRuleExclusionDirectives(exclusions)
	if err != nil {
		t.Fatalf("buildWafRuleExclusionDirectives() error = %v", err)
	}

	expectedFragments := []string{
		"SecRuleRemoveById 920350",
		`REQUEST_HEADERS:Host "@streq app.example.com"`,
		`REQUEST_URI "@beginsWith /api/login"`,
		`REQUEST_METHOD "@streq POST"`,
		"ctl:ruleRemoveByTag=attack-sqli",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(directives, fragment) {
			t.Fatalf("expected directives contains %q, got: %s", fragment, directives)
		}
	}
}

func TestEnsureNoPolicyBindingConflictsNoConflict(t *testing.T) {
	db, mock, cleanup := newPolicyScopeMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT .* FROM "waf_policy_bindings"`).WillReturnRows(
		sqlmock.NewRows([]string{"scope_type", "host", "path", "method", "priority", "count"}),
	)

	if err := ensureNoPolicyBindingConflicts(db); err != nil {
		t.Fatalf("ensureNoPolicyBindingConflicts() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestEnsureNoPolicyBindingConflictsHasConflict(t *testing.T) {
	db, mock, cleanup := newPolicyScopeMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT .* FROM "waf_policy_bindings"`).WillReturnRows(
		sqlmock.NewRows([]string{"scope_type", "host", "path", "method", "priority", "count"}).
			AddRow("route", "app.example.com", "/api", "GET", int64(100), int64(2)),
	)

	err := ensureNoPolicyBindingConflicts(db)
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if !strings.Contains(err.Error(), "policy binding conflicts found") {
		t.Fatalf("expected conflict error message, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func newPolicyScopeMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
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
