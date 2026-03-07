package caddy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/svc"
	"logflux/internal/types"
)

func TestGetWafIntegrationStatus(t *testing.T) {
	gdb, mock, cleanup := newWafIntegrationMockDB(t)
	defer cleanup()

	now := time.Now()
	config := "{\n  admin :2019\n  order coraza_waf first\n}\n\n" + renderWafProtectSnippet("\n") + "\nexample.com {\n  import waf_protect\n  reverse_proxy localhost:8080\n}\n"
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(caddyServerRows(now, "http://127.0.0.1:2019", config))

	logic := NewGetWafIntegrationStatusLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetWafIntegrationStatus()
	if err != nil {
		t.Fatalf("GetWafIntegrationStatus() error = %v", err)
	}
	if resp == nil || !resp.Integrated {
		t.Fatalf("expected integrated response, got %+v", resp)
	}
	if len(resp.ImportedSites) != 1 || resp.ImportedSites[0] != "example.com" {
		t.Fatalf("unexpected imported sites: %+v", resp.ImportedSites)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestApplyWafIntegrationDryRun(t *testing.T) {
	gdb, mock, cleanup := newWafIntegrationMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(caddyServerRows(now, "http://127.0.0.1:2019", integrationBaseConfig))

	logic := NewApplyWafIntegrationLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.ApplyWafIntegration(&types.WafIntegrationApplyReq{
		Enabled:       true,
		SiteAddresses: []string{"example.com"},
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("ApplyWafIntegration() error = %v", err)
	}
	if resp == nil || !resp.Changed {
		t.Fatalf("expected changed dry-run response, got %+v", resp)
	}
	if !strings.Contains(resp.Config, "import waf_protect") {
		t.Fatalf("expected preview config to contain import waf_protect")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestApplyWafIntegrationPersist(t *testing.T) {
	var requests []string
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		requests = append(requests, r.URL.Path+"\n"+string(body))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newWafIntegrationMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(caddyServerRows(now, caddyMock.URL, integrationBaseConfig))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "caddy_servers" SET`)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "caddy_config_history"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	logic := NewApplyWafIntegrationLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.ApplyWafIntegration(&types.WafIntegrationApplyReq{
		Enabled:       true,
		SiteAddresses: []string{"example.com"},
	})
	if err != nil {
		t.Fatalf("ApplyWafIntegration() error = %v", err)
	}
	if resp == nil || !resp.Changed {
		t.Fatalf("expected changed response, got %+v", resp)
	}
	if len(requests) != 2 {
		t.Fatalf("expected adapt and load requests, got %d", len(requests))
	}
	if !strings.Contains(requests[0], "/adapt") || !strings.Contains(requests[1], "/load") {
		t.Fatalf("unexpected request sequence: %+v", requests)
	}
	if !strings.Contains(requests[1], "import waf_protect") {
		t.Fatalf("expected load payload to contain import waf_protect")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func newWafIntegrationMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
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
