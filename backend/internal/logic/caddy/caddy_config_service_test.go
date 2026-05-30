package caddy

import (
	"context"
	"encoding/json"
	"io"
	caddymodel "logflux/model/caddy"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"logflux/internal/svc"
	"logflux/internal/types"
)

func TestCaddyConfigServicePrepareQuick(t *testing.T) {
	modules := mustCaddyModules(t, caddyFormModel{
		SchemaVersion: 1,
		Global: caddyGlobal{
			Raw: "{\n  admin :2019\n}",
		},
		Upstreams: []caddyUpstream{},
		Sites: []caddySite{
			{
				ID:      "site-1",
				Name:    "默认站点",
				Enabled: true,
				Domains: []string{":80"},
				Encode:  []string{"gzip"},
				TLS:     &caddyTLS{Mode: "auto"},
				Routes: []caddyRoute{
					{
						ID:      "health",
						Name:    "@api_health",
						Enabled: true,
						Match:   caddyMatch{Path: []string{"/api/health"}},
						Handles: []caddyHandle{
							{ID: "h1", Type: "respond", Enabled: true, Body: "OK", Status: 200},
						},
					},
					{
						ID:      "api",
						Name:    "@api",
						Enabled: true,
						Match:   caddyMatch{Path: []string{"/api/*"}},
						Handles: []caddyHandle{
							{ID: "h2", Type: "reverse_proxy", Enabled: true, Upstream: "localhost:8888"},
						},
					},
					{
						ID:      "spa",
						Name:    "默认路由",
						Enabled: true,
						Handles: []caddyHandle{
							{ID: "h3", Type: "rewrite", Enabled: true, URI: "/index.html"},
							{ID: "h4", Type: "file_server", Enabled: true, Root: "/app/frontend"},
						},
					},
				},
			},
		},
	})

	result, err := newCaddyConfigService().Prepare(caddyConfigPrepareInput{
		Mode:    caddyConfigModeQuick,
		Modules: modules,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	for _, expected := range []string{
		"admin :2019",
		"encode gzip",
		"respond \"OK\" 200",
		"reverse_proxy localhost:8888",
		"root * /app/frontend",
		"file_server",
	} {
		if !strings.Contains(result.Config, expected) {
			t.Fatalf("expected config to contain %q, got:\n%s", expected, result.Config)
		}
	}
	if result.Modules == "" || result.Modules == emptyModulesJSON {
		t.Fatalf("expected normalized modules, got %q", result.Modules)
	}
}

func TestCaddyConfigServicePrepareQuickUsesMergedConfig(t *testing.T) {
	modules := mustCaddyModules(t, caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{Raw: "# 通用配置片段（Snippets）"},
		Upstreams:     []caddyUpstream{},
		Sites: []caddySite{
			{
				ID:      "site-1",
				Name:    "app",
				Enabled: true,
				Domains: []string{"example.com"},
				Routes: []caddyRoute{
					{
						ID:      "r1",
						Name:    "默认路由",
						Enabled: true,
						Handles: []caddyHandle{
							{ID: "h1", Type: "reverse_proxy", Enabled: true, Upstream: "localhost:8888"},
						},
					},
				},
			},
		},
	})
	mergedConfig := `# 通用配置片段（Snippets）

(common_headers) {
  header X-Test preserved
}

example.com {
  import common_headers
  reverse_proxy localhost:8888
}`

	result, err := newCaddyConfigService().Prepare(caddyConfigPrepareInput{
		Mode:    caddyConfigModeQuick,
		Config:  mergedConfig,
		Modules: modules,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	for _, expected := range []string{
		"(common_headers)",
		"header X-Test preserved",
		"import common_headers",
		"reverse_proxy localhost:8888",
	} {
		if !strings.Contains(result.Config, expected) {
			t.Fatalf("expected merged config to contain %q, got:\n%s", expected, result.Config)
		}
	}
}

func TestCaddyConfigServicePrepareQuickCompletesWafProtectSnippet(t *testing.T) {
	modules := mustCaddyModules(t, caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{},
		Upstreams:     []caddyUpstream{},
		Sites: []caddySite{
			{
				ID:      "site-1",
				Name:    "app",
				Enabled: true,
				Domains: []string{"example.com"},
				Imports: []string{"waf_protect"},
				Routes: []caddyRoute{
					{
						ID:      "r1",
						Name:    "默认路由",
						Enabled: true,
						Handles: []caddyHandle{
							{ID: "h1", Type: "reverse_proxy", Enabled: true, Upstream: "localhost:8888"},
						},
					},
				},
			},
		},
	})
	mergedConfig := `example.com {
  import waf_protect
  reverse_proxy localhost:8888
}`

	result, err := newCaddyConfigService().Prepare(caddyConfigPrepareInput{
		Mode:    caddyConfigModeQuick,
		Config:  mergedConfig,
		Modules: modules,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	for _, expected := range []string{
		"order coraza_waf first",
		"(waf_protect)",
		"coraza_waf",
		"directives `",
		"import waf_protect",
		"reverse_proxy localhost:8888",
	} {
		if !strings.Contains(result.Config, expected) {
			t.Fatalf("expected completed WAF config to contain %q, got:\n%s", expected, result.Config)
		}
	}
}

func TestCaddyConfigServiceValidateQuick(t *testing.T) {
	modules := mustCaddyModules(t, caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{},
		Upstreams: []caddyUpstream{
			{Name: "app", Targets: []string{}},
		},
		Sites: []caddySite{
			{
				ID:      "site-1",
				Name:    "bad",
				Enabled: true,
				Domains: []string{"bad domain"},
				Routes: []caddyRoute{
					{
						ID:      "r1",
						Name:    "route",
						Enabled: true,
						Match:   caddyMatch{Path: []string{"not-path"}},
						Handles: []caddyHandle{{ID: "h1", Type: "reverse_proxy", Enabled: true}},
					},
				},
			},
		},
	})

	_, err := newCaddyConfigService().Prepare(caddyConfigPrepareInput{
		Mode:    caddyConfigModeQuick,
		Modules: modules,
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	for _, expected := range []string{"至少配置一个目标", "域名格式不合法", "Path 格式不合法", "未选择上游"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to contain %q, got %v", expected, err)
		}
	}
}

func TestResolveCaddyConfigModulesForUpdateClearsRawSnapshot(t *testing.T) {
	request := mustCaddyModules(t, caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{Raw: "{"},
		Sites: []caddySite{
			{
				ID:      "site-1",
				Name:    "app",
				Enabled: true,
				Domains: []string{"example.com"},
				Routes: []caddyRoute{
					{
						ID:      "route-1",
						Name:    "默认路由",
						Enabled: true,
						Handles: []caddyHandle{{ID: "h1", Type: "reverse_proxy", Enabled: true, Upstream: "localhost:8888"}},
					},
				},
			},
		},
	})
	if got := resolveCaddyConfigModulesForUpdate(caddyConfigModeRaw, ""); got != emptyModulesJSON {
		t.Fatalf("expected raw update to clear modules, got %q", got)
	}
	if got := resolveCaddyConfigModulesForUpdate(caddyConfigModeRaw, request); got != emptyModulesJSON {
		t.Fatalf("expected raw update to ignore request modules, got %q", got)
	}
	if got := resolveCaddyConfigModulesForUpdate(caddyConfigModeQuick, request); got != request {
		t.Fatalf("expected structured update to use request modules, got %q", got)
	}
}

func TestCaddyConfigFromServerReturnsEmptyModulesWhenConfigMissing(t *testing.T) {
	server := &caddymodel.CaddyServer{
		Config:  "",
		Modules: mustCaddyModules(t, caddyFormModel{SchemaVersion: 1, Global: caddyGlobal{Raw: "{x}"}}),
	}
	_, modules := caddyConfigFromServer(server)
	if modules != emptyModulesJSON {
		t.Fatalf("expected empty modules when config is missing, got %q", modules)
	}
}

func TestLoadCurrentCaddyConfigUsesEmptyModulesForLocalFallback(t *testing.T) {
	tmpDir := t.TempDir()
	caddyfilePath := filepath.Join(tmpDir, "Caddyfile")
	rawConfig := "example.com {\n  reverse_proxy localhost:8888\n}\n"
	if err := os.WriteFile(caddyfilePath, []byte(rawConfig), 0o644); err != nil {
		t.Fatalf("write temp caddyfile: %v", err)
	}

	originalPath := localCaddyfilePath
	localCaddyfilePath = caddyfilePath
	defer func() {
		localCaddyfilePath = originalPath
	}()

	server := &caddymodel.CaddyServer{
		Type:    "local",
		Config:  "",
		Modules: mustCaddyModules(t, caddyFormModel{SchemaVersion: 1, Global: caddyGlobal{Raw: "{\"stale\":true}"}}),
	}

	config, modules, err := loadCurrentCaddyConfig(server)
	if err != nil {
		t.Fatalf("loadCurrentCaddyConfig() error = %v", err)
	}
	if strings.TrimSpace(config) != strings.TrimSpace(rawConfig) {
		t.Fatalf("expected local file config, got %q", config)
	}
	if modules != emptyModulesJSON {
		t.Fatalf("expected empty modules for local fallback, got %q", modules)
	}
}

func TestGetCaddyConfigUsesEmptyModulesForLocalFallback(t *testing.T) {
	tmpDir := t.TempDir()
	caddyfilePath := filepath.Join(tmpDir, "Caddyfile")
	rawConfig := "example.com {\n  reverse_proxy localhost:8888\n}\n"
	if err := os.WriteFile(caddyfilePath, []byte(rawConfig), 0o644); err != nil {
		t.Fatalf("write temp caddyfile: %v", err)
	}

	originalPath := localCaddyfilePath
	localCaddyfilePath = caddyfilePath
	defer func() {
		localCaddyfilePath = originalPath
	}()

	gdb, mock, cleanup := newWafIntegrationMockDB(t)
	defer cleanup()

	now := time.Now()
	modules := mustCaddyModules(t, caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{Raw: "{\"stale\":true}"},
		Sites: []caddySite{{
			ID:      "site-1",
			Name:    "app",
			Enabled: true,
			Domains: []string{"example.com"},
			Routes: []caddyRoute{{
				ID:      "r1",
				Name:    "默认路由",
				Enabled: true,
				Handles: []caddyHandle{{ID: "h1", Type: "reverse_proxy", Enabled: true, Upstream: "localhost:8888"}},
			}},
		}},
	})
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(caddyServerRowsWithModules(now, "http://127.0.0.1:2019", "", modules))

	logic := NewGetCaddyConfigLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.GetCaddyConfig(&types.CaddyConfigReq{ServerId: 1})
	if err != nil {
		t.Fatalf("GetCaddyConfig() error = %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response")
	}
	if strings.TrimSpace(resp.Config) != strings.TrimSpace(rawConfig) {
		t.Fatalf("expected local file config, got %q", resp.Config)
	}
	if resp.Modules != emptyModulesJSON {
		t.Fatalf("expected empty modules for local fallback, got %q", resp.Modules)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestPreviewCaddyConfigOnlyAdapts(t *testing.T) {
	var paths []string
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		paths = append(paths, r.URL.Path+"\n"+string(body))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newWafIntegrationMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(caddyServerRows(now, caddyMock.URL, integrationBaseConfig))

	modules := mustCaddyModules(t, caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{},
		Upstreams:     []caddyUpstream{},
		Sites: []caddySite{
			{
				ID:      "site-1",
				Name:    "app",
				Enabled: true,
				Domains: []string{"example.com"},
				Routes: []caddyRoute{
					{
						ID:      "r1",
						Name:    "默认路由",
						Enabled: true,
						Handles: []caddyHandle{
							{ID: "h1", Type: "reverse_proxy", Enabled: true, Upstream: "localhost:8888"},
						},
					},
				},
			},
		},
	})

	logic := NewPreviewCaddyConfigLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.PreviewCaddyConfig(&types.CaddyConfigPreviewReq{
		ServerId: 1,
		Mode:     caddyConfigModeQuick,
		Modules:  modules,
	})
	if err != nil {
		t.Fatalf("PreviewCaddyConfig() error = %v", err)
	}
	if resp == nil || !resp.Valid {
		t.Fatalf("expected valid preview, got %+v", resp)
	}
	if len(paths) != 1 || !strings.Contains(paths[0], "/adapt") {
		t.Fatalf("expected only adapt request, got %+v", paths)
	}
	if strings.Contains(paths[0], "/load") {
		t.Fatalf("preview must not load config: %+v", paths)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func mustCaddyModules(t *testing.T, model caddyFormModel) string {
	t.Helper()
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal modules: %v", err)
	}
	return string(raw)
}

func caddyServerRowsWithModules(now time.Time, url, config, modules string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at",
		"name", "url", "token", "type", "username", "password",
		"config", "modules",
	}).AddRow(
		uint(1), now, now,
		"local-default", url, "", "local", "", "",
		config, modules,
	)
}
