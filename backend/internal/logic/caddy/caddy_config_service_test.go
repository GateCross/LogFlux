package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	caddymodel "logflux/model/caddy"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"regexp"
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

func TestCaddyConfigServicePrepareRawRejectsCommentOnly(t *testing.T) {
	_, err := newCaddyConfigService().Prepare(caddyConfigPrepareInput{
		Mode:   caddyConfigModeRaw,
		Config: "# LogFlux 尚未接管 Caddy 配置。\n# 请粘贴当前 Caddyfile 后保存。",
	})
	if err == nil {
		t.Fatalf("expected comment-only raw config error")
	}
	if !strings.Contains(err.Error(), "Caddy 配置不能为空") {
		t.Fatalf("expected empty config error, got %v", err)
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

func TestResolveCaddyConfigModulesKeepsRawSnapshot(t *testing.T) {
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
	existing := mustCaddyModules(t, caddyFormModel{SchemaVersion: 1, Global: caddyGlobal{Raw: "{existing}"}})
	if got := resolveCaddyConfigModules(caddyConfigModeRaw, "", existing); got != existing {
		t.Fatalf("expected raw update to keep existing modules, got %q", got)
	}
	if got := resolveCaddyConfigModules(caddyConfigModeRaw, request, existing); got != request {
		t.Fatalf("expected raw update with request modules to use request modules, got %q", got)
	}
	if got := resolveCaddyConfigModules(caddyConfigModeQuick, request, existing); got != request {
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

func TestLoadCurrentCaddyConfigRequiresManagedConfig(t *testing.T) {
	server := &caddymodel.CaddyServer{
		Type:    "local",
		Config:  "",
		Modules: mustCaddyModules(t, caddyFormModel{SchemaVersion: 1, Global: caddyGlobal{Raw: "{\"stale\":true}"}}),
	}

	config, modules, err := loadCurrentCaddyConfig(server)
	if err == nil {
		t.Fatalf("expected unmanaged config error")
	}
	if config != "" {
		t.Fatalf("expected empty config when unmanaged, got %q", config)
	}
	if modules != emptyModulesJSON {
		t.Fatalf("expected empty modules when unmanaged, got %q", modules)
	}
	if !strings.Contains(err.Error(), "尚未由 LogFlux 接管") {
		t.Fatalf("expected unmanaged error, got %v", err)
	}
}

func TestGetCaddyConfigReturnsUnmanagedPrompt(t *testing.T) {
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
	if !strings.Contains(resp.Config, "尚未接管 Caddy 配置") {
		t.Fatalf("expected unmanaged prompt, got %q", resp.Config)
	}
	if resp.Modules != emptyModulesJSON {
		t.Fatalf("expected empty modules when unmanaged, got %q", resp.Modules)
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
	if !resp.Valid {
		t.Fatalf("expected valid preview, got %+v", resp)
	}
	// Req 6.2: Preview 仅 /adapt，actions 明确声明未 /load
	joinedActions := strings.Join(resp.Actions, " ")
	if !strings.Contains(joinedActions, "预览未执行 Caddy /load") && !strings.Contains(joinedActions, "未执行 Caddy /load") {
		t.Fatalf("expected preview actions to state no /load, got %+v", resp.Actions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

// TestUpdateCaddyConfigUsesAdaptThenLoadAndHistory locks Apply_Path (Req 6.3):
// Update 走 adapt → load，并写入 config history；与 Preview 仅 adapt 形成对照。
func TestUpdateCaddyConfigUsesAdaptThenLoadAndHistory(t *testing.T) {
	var paths []string
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Update 后异步 log-source sync 可能 GET /config/；只记录 Admin 写路径。
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		paths = append(paths, r.URL.Path)
		_ = body
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newWafIntegrationMockDB(t)
	defer cleanup()

	now := time.Now()
	// 已有配置 → apply 走 load 后再 persist 分支
	mock.ExpectQuery(`SELECT .* FROM "caddy_servers"`).WillReturnRows(
		caddyServerRows(now, caddyMock.URL, integrationBaseConfig),
	)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "caddy_servers" SET`)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "caddy_config_history"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))
	mock.ExpectCommit()

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
							{
								ID:       "h1",
								Type:     "reverse_proxy",
								Enabled:  true,
								Upstream: "localhost:8888",
								LBPolicy: "least_conn",
								HealthCheck: &caddyHealthCheck{
									Path:     "/healthz",
									Interval: "10s",
									Timeout:  "2s",
								},
							},
						},
					},
				},
			},
		},
	})

	logic := NewUpdateCaddyConfigLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	resp, err := logic.UpdateCaddyConfig(&types.CaddyConfigUpdateReq{
		ServerId: 1,
		Mode:     caddyConfigModeQuick,
		Modules:  modules,
	})
	if err != nil {
		t.Fatalf("UpdateCaddyConfig() error = %v", err)
	}
	if resp == nil || resp.Code != 200 {
		t.Fatalf("expected success response, got %+v", resp)
	}

	// 给异步 log sync 一点时间，避免干扰主路径断言
	time.Sleep(50 * time.Millisecond)

	var sawAdapt, sawLoad bool
	for _, p := range paths {
		if strings.Contains(p, "/adapt") {
			sawAdapt = true
		}
		if strings.Contains(p, "/load") {
			sawLoad = true
		}
	}
	if !sawAdapt {
		t.Fatalf("Update must call /adapt first, paths=%v", paths)
	}
	if !sawLoad {
		t.Fatalf("Update must call /load (unique Apply_Path hot-load), paths=%v", paths)
	}
	// 顺序：adapt 先于 load
	adaptIdx, loadIdx := -1, -1
	for i, p := range paths {
		if adaptIdx < 0 && strings.Contains(p, "/adapt") {
			adaptIdx = i
		}
		if loadIdx < 0 && strings.Contains(p, "/load") {
			loadIdx = i
		}
	}
	if adaptIdx < 0 || loadIdx < 0 || adaptIdx > loadIdx {
		t.Fatalf("expected /adapt before /load, paths=%v", paths)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

// TestSharedHealthContractCases locks FE/BE 同一语义契约的后端侧（Req 3.6 / 6.x）：
// 与 frontend caddy-config-blocks.test.ts 中 shared contract 用例输入一致。
func TestSharedHealthContractCases(t *testing.T) {
	type sharedCase struct {
		name        string
		path        string
		interval    string
		timeout     string
		lbPolicy    string
		wantURI     string
		wantInt     string
		wantTimeout string
		wantLB      string
	}
	cases := []sharedCase{
		{
			name:        "full health + least_conn",
			path:        "/healthz",
			interval:    "10s",
			timeout:     "2s",
			lbPolicy:    "least_conn",
			wantURI:     "/healthz",
			wantInt:     "10s",
			wantTimeout: "2s",
			wantLB:      "least_conn",
		},
		{
			name:     "path only + ip_hash",
			path:     "/ready",
			lbPolicy: "ip_hash",
			wantURI:  "/ready",
			wantLB:   "ip_hash",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hc := &caddyHealthCheck{Path: tc.path}
			if tc.interval != "" {
				hc.Interval = tc.interval
			}
			if tc.timeout != "" {
				hc.Timeout = tc.timeout
			}
			out := renderReverseProxySnippet(t, reverseProxySnippetOpts{
				upstream:    "127.0.0.1:8080",
				healthCheck: hc,
				lbPolicy:    tc.lbPolicy,
			})
			if !strings.Contains(out, "health_uri "+tc.wantURI) {
				t.Fatalf("expected health_uri %q in:\n%s", tc.wantURI, out)
			}
			if tc.wantInt != "" {
				if !strings.Contains(out, "health_interval "+tc.wantInt) {
					t.Fatalf("expected health_interval %q in:\n%s", tc.wantInt, out)
				}
			} else if strings.Contains(out, "health_interval") {
				t.Fatalf("did not expect health_interval in:\n%s", out)
			}
			if tc.wantTimeout != "" {
				if !strings.Contains(out, "health_timeout "+tc.wantTimeout) {
					t.Fatalf("expected health_timeout %q in:\n%s", tc.wantTimeout, out)
				}
			} else if strings.Contains(out, "health_timeout") {
				t.Fatalf("did not expect health_timeout in:\n%s", out)
			}
			if !strings.Contains(out, "lb_policy "+tc.wantLB) {
				t.Fatalf("expected lb_policy %q in:\n%s", tc.wantLB, out)
			}
		})
	}
}

// Property 3: Health 指令与模型一致
// 对任意启用且 path 有效的 Health_Check，生成的 Caddyfile 须包含对应
// health_uri（以及已设置的 health_interval/health_timeout）；禁用或全空时，
// 三条指令均不得出现。
// **验证：Requirements 3.1, 3.2**
func TestProperty3_HealthDirectivesMatchModel(t *testing.T) {
	type caseInput struct {
		name        string
		healthCheck *caddyHealthCheck
		lbPolicy    string
		wantURI     string
		wantInt     string
		wantTimeout string
		wantLB      string
		omitHealth  bool
	}

	cases := []caseInput{
		{
			name:        "enabled full health + least_conn",
			healthCheck: &caddyHealthCheck{Path: "/healthz", Interval: "10s", Timeout: "2s"},
			lbPolicy:    "least_conn",
			wantURI:     "/healthz",
			wantInt:     "10s",
			wantTimeout: "2s",
			wantLB:      "least_conn",
		},
		{
			name:        "enabled path only",
			healthCheck: &caddyHealthCheck{Path: "/ready"},
			wantURI:     "/ready",
		},
		{
			name:        "enabled path + interval only",
			healthCheck: &caddyHealthCheck{Path: "/live", Interval: "5s"},
			wantURI:     "/live",
			wantInt:     "5s",
		},
		{
			name:        "enabled path + timeout only",
			healthCheck: &caddyHealthCheck{Path: "/ok", Timeout: "1s"},
			wantURI:     "/ok",
			wantTimeout: "1s",
		},
		{
			name:       "disabled nil health",
			omitHealth: true,
		},
		{
			name:        "disabled empty path",
			healthCheck: &caddyHealthCheck{Path: ""},
			omitHealth:  true,
		},
		{
			name:        "disabled whitespace path",
			healthCheck: &caddyHealthCheck{Path: "   ", Interval: "", Timeout: ""},
			omitHealth:  true,
		},
		{
			name:        "disabled empty path with interval noise",
			healthCheck: &caddyHealthCheck{Path: "", Interval: "10s", Timeout: "1s"},
			omitHealth:  true,
		},
		{
			name:       "lb only without health",
			lbPolicy:   "ip_hash",
			wantLB:     "ip_hash",
			omitHealth: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := renderReverseProxySnippet(t, reverseProxySnippetOpts{
				upstream:    "127.0.0.1:8080",
				healthCheck: tc.healthCheck,
				lbPolicy:    tc.lbPolicy,
			})
			if tc.omitHealth {
				assertNoHealthDirectives(t, out)
			} else {
				if !strings.Contains(out, "health_uri "+tc.wantURI) {
					t.Fatalf("expected health_uri %q in:\n%s", tc.wantURI, out)
				}
				if tc.wantInt != "" {
					if !strings.Contains(out, "health_interval "+tc.wantInt) {
						t.Fatalf("expected health_interval %q in:\n%s", tc.wantInt, out)
					}
				} else if strings.Contains(out, "health_interval") {
					t.Fatalf("did not expect health_interval in:\n%s", out)
				}
				if tc.wantTimeout != "" {
					if !strings.Contains(out, "health_timeout "+tc.wantTimeout) {
						t.Fatalf("expected health_timeout %q in:\n%s", tc.wantTimeout, out)
					}
				} else if strings.Contains(out, "health_timeout") {
					t.Fatalf("did not expect health_timeout in:\n%s", out)
				}
			}
			if tc.wantLB != "" {
				if !strings.Contains(out, "lb_policy "+tc.wantLB) {
					t.Fatalf("expected lb_policy %q in:\n%s", tc.wantLB, out)
				}
			}
			if tc.omitHealth && tc.wantLB == "" {
				if strings.Contains(out, "reverse_proxy 127.0.0.1:8080 {") {
					t.Fatalf("expected single-line reverse_proxy without empty block, got:\n%s", out)
				}
				if !strings.Contains(out, "reverse_proxy 127.0.0.1:8080") {
					t.Fatalf("expected reverse_proxy line, got:\n%s", out)
				}
			}
		})
	}

	// Property 3 多轮覆盖（设计：≥100）
	rng := rand.New(rand.NewSource(42))
	units := []string{"ms", "s", "m", "h"}
	policies := []string{"", "round_robin", "least_conn", "ip_hash"}
	for i := 0; i < 100; i++ {
		enabled := rng.Intn(2) == 0
		var hc *caddyHealthCheck
		var wantURI, wantInt, wantTimeout string
		if enabled {
			wantURI = "/" + randomHealthSegment(rng)
			hc = &caddyHealthCheck{Path: wantURI}
			if rng.Intn(2) == 0 {
				wantInt = fmt.Sprintf("%d%s", 1+rng.Intn(60), units[rng.Intn(len(units))])
				hc.Interval = wantInt
			}
			if rng.Intn(2) == 0 {
				wantTimeout = fmt.Sprintf("%d%s", 1+rng.Intn(30), units[rng.Intn(len(units))])
				hc.Timeout = wantTimeout
			}
		} else {
			switch rng.Intn(4) {
			case 0:
				hc = nil
			case 1:
				hc = &caddyHealthCheck{Path: ""}
			case 2:
				hc = &caddyHealthCheck{Path: "   "}
			default:
				hc = &caddyHealthCheck{Path: "", Interval: "10s", Timeout: "1s"}
			}
		}
		lb := policies[rng.Intn(len(policies))]
		out := renderReverseProxySnippet(t, reverseProxySnippetOpts{
			upstream:    "127.0.0.1:8080",
			healthCheck: hc,
			lbPolicy:    lb,
		})
		if enabled {
			if !strings.Contains(out, "health_uri "+wantURI) {
				t.Fatalf("iter %d: expected health_uri %q in:\n%s", i, wantURI, out)
			}
			if wantInt != "" && !strings.Contains(out, "health_interval "+wantInt) {
				t.Fatalf("iter %d: expected health_interval %q in:\n%s", i, wantInt, out)
			}
			if wantInt == "" && strings.Contains(out, "health_interval") {
				t.Fatalf("iter %d: unexpected health_interval in:\n%s", i, out)
			}
			if wantTimeout != "" && !strings.Contains(out, "health_timeout "+wantTimeout) {
				t.Fatalf("iter %d: expected health_timeout %q in:\n%s", i, wantTimeout, out)
			}
			if wantTimeout == "" && strings.Contains(out, "health_timeout") {
				t.Fatalf("iter %d: unexpected health_timeout in:\n%s", i, out)
			}
		} else {
			assertNoHealthDirectives(t, out)
		}
		if lb != "" && !strings.Contains(out, "lb_policy "+lb) {
			t.Fatalf("iter %d: expected lb_policy %q in:\n%s", i, lb, out)
		}
	}
}

// Property 4: handle 覆盖 pool health
// 对任意同时具备 handle 级与 pool 级有效 Health_Check 的 reverse_proxy，
// 生成结果使用 handle 级 path（及 interval/timeout）。
// **验证：Requirements 3.4**
func TestProperty4_HandleHealthOverridesPool(t *testing.T) {
	poolHC := &caddyHealthCheck{Path: "/pool-health", Interval: "30s", Timeout: "5s"}
	handleHC := &caddyHealthCheck{Path: "/handle-health", Interval: "10s", Timeout: "2s"}

	t.Run("handle overrides pool", func(t *testing.T) {
		out := renderReverseProxySnippet(t, reverseProxySnippetOpts{
			upstream:     "app-pool",
			poolName:     "app-pool",
			poolTargets:  []string{"10.0.0.1:9000", "10.0.0.2:9000"},
			poolHealth:   poolHC,
			poolLBPolicy: "round_robin",
			healthCheck:  handleHC,
			lbPolicy:     "least_conn",
		})
		if !strings.Contains(out, "health_uri /handle-health") {
			t.Fatalf("expected handle health_uri, got:\n%s", out)
		}
		if strings.Contains(out, "health_uri /pool-health") {
			t.Fatalf("must not emit pool health_uri when handle wins:\n%s", out)
		}
		if !strings.Contains(out, "health_interval 10s") || !strings.Contains(out, "health_timeout 2s") {
			t.Fatalf("expected handle interval/timeout, got:\n%s", out)
		}
		if strings.Contains(out, "health_interval 30s") || strings.Contains(out, "health_timeout 5s") {
			t.Fatalf("must not emit pool interval/timeout when handle wins:\n%s", out)
		}
		if !strings.Contains(out, "lb_policy least_conn") {
			t.Fatalf("expected handle lb_policy, got:\n%s", out)
		}
		if strings.Contains(out, "lb_policy round_robin") {
			t.Fatalf("must not emit pool lb when handle wins:\n%s", out)
		}
		if !strings.Contains(out, "reverse_proxy 10.0.0.1:9000 10.0.0.2:9000 {") {
			t.Fatalf("expected pool targets expanded, got:\n%s", out)
		}
	})

	t.Run("pool used when handle health empty", func(t *testing.T) {
		out := renderReverseProxySnippet(t, reverseProxySnippetOpts{
			upstream:     "app-pool",
			poolName:     "app-pool",
			poolTargets:  []string{"10.0.0.1:9000"},
			poolHealth:   poolHC,
			poolLBPolicy: "ip_hash",
			healthCheck:  &caddyHealthCheck{Path: ""},
		})
		if !strings.Contains(out, "health_uri /pool-health") {
			t.Fatalf("expected pool health_uri fallback, got:\n%s", out)
		}
		if !strings.Contains(out, "health_interval 30s") || !strings.Contains(out, "health_timeout 5s") {
			t.Fatalf("expected pool interval/timeout, got:\n%s", out)
		}
		if !strings.Contains(out, "lb_policy ip_hash") {
			t.Fatalf("expected pool lb_policy fallback, got:\n%s", out)
		}
	})

	// Property 4 多轮覆盖
	rng := rand.New(rand.NewSource(7))
	units := []string{"ms", "s", "m", "h"}
	for i := 0; i < 100; i++ {
		hPath := "/h-" + randomHealthSegment(rng)
		pPath := "/p-" + randomHealthSegment(rng)
		hInt := fmt.Sprintf("%d%s", 1+rng.Intn(20), units[rng.Intn(len(units))])
		pInt := fmt.Sprintf("%d%s", 21+rng.Intn(20), units[rng.Intn(len(units))])
		hTo := fmt.Sprintf("%d%s", 1+rng.Intn(10), units[rng.Intn(len(units))])
		pTo := fmt.Sprintf("%d%s", 11+rng.Intn(10), units[rng.Intn(len(units))])
		out := renderReverseProxySnippet(t, reverseProxySnippetOpts{
			upstream:    "pool",
			poolName:    "pool",
			poolTargets: []string{"127.0.0.1:9000"},
			poolHealth:  &caddyHealthCheck{Path: pPath, Interval: pInt, Timeout: pTo},
			healthCheck: &caddyHealthCheck{Path: hPath, Interval: hInt, Timeout: hTo},
		})
		if !strings.Contains(out, "health_uri "+hPath) {
			t.Fatalf("iter %d: expected handle path %q in:\n%s", i, hPath, out)
		}
		if strings.Contains(out, "health_uri "+pPath) {
			t.Fatalf("iter %d: pool path must not win, got:\n%s", i, out)
		}
		if !strings.Contains(out, "health_interval "+hInt) || !strings.Contains(out, "health_timeout "+hTo) {
			t.Fatalf("iter %d: expected handle interval/timeout, got:\n%s", i, out)
		}
	}
}

func TestValidateCaddyHealthCheckFields(t *testing.T) {
	errs := validateCaddyHealthCheckFields(&caddyHealthCheck{Path: "healthz", Interval: "10s"}, "健康检查")
	if !containsSubstring(errs, "路径") || !containsSubstring(errs, "/") {
		t.Fatalf("expected path slash error, got %v", errs)
	}
	errs = validateCaddyHealthCheckFields(&caddyHealthCheck{Path: "/ok", Interval: "10", Timeout: "fast"}, "健康检查")
	if !containsSubstring(errs, "间隔") || !containsSubstring(errs, "超时") {
		t.Fatalf("expected duration errors, got %v", errs)
	}
	errs = validateCaddyHealthCheckFields(&caddyHealthCheck{Path: "/ok", Interval: "10s", Timeout: "1s"}, "健康检查")
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestPrepareQuickRejectsInvalidHealthCheck(t *testing.T) {
	modules := mustCaddyModules(t, caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{},
		Upstreams: []caddyUpstream{
			{
				Name:        "app",
				Targets:     []string{"127.0.0.1:8080"},
				HealthCheck: &caddyHealthCheck{Path: "no-slash", Interval: "1s"},
			},
		},
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
							{
								ID:          "h1",
								Type:        "reverse_proxy",
								Enabled:     true,
								Upstream:    "app",
								HealthCheck: &caddyHealthCheck{Path: "also-bad", Timeout: "fast"},
							},
						},
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
		t.Fatalf("expected health validation error")
	}
	msg := err.Error()
	for _, expected := range []string{"健康检查", "路径", "/"} {
		if !strings.Contains(msg, expected) {
			t.Fatalf("expected error to contain %q, got %v", expected, err)
		}
	}
}

func TestRenderCaddyReverseProxyWithTransportAndHealth(t *testing.T) {
	out := renderReverseProxySnippet(t, reverseProxySnippetOpts{
		upstream:              "127.0.0.1:8443",
		healthCheck:           &caddyHealthCheck{Path: "/health"},
		lbPolicy:              "round_robin",
		transportProtocol:     "http",
		tlsInsecureSkipVerify: true,
	})
	for _, expected := range []string{
		"reverse_proxy 127.0.0.1:8443 {",
		"lb_policy round_robin",
		"health_uri /health",
		"transport http {",
		"tls_insecure_skip_verify",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected %q in:\n%s", expected, out)
		}
	}
}

type reverseProxySnippetOpts struct {
	upstream              string
	healthCheck           *caddyHealthCheck
	lbPolicy              string
	transportProtocol     string
	tlsInsecureSkipVerify bool
	poolName              string
	poolTargets           []string
	poolHealth            *caddyHealthCheck
	poolLBPolicy          string
}

func renderReverseProxySnippet(t *testing.T, opts reverseProxySnippetOpts) string {
	t.Helper()
	upstreams := []caddyUpstream{}
	if opts.poolName != "" {
		upstreams = append(upstreams, caddyUpstream{
			Name:        opts.poolName,
			Targets:     opts.poolTargets,
			LBPolicy:    opts.poolLBPolicy,
			HealthCheck: opts.poolHealth,
		})
	}
	model := caddyFormModel{
		SchemaVersion: 1,
		Global:        caddyGlobal{},
		Upstreams:     upstreams,
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
							{
								ID:                    "h1",
								Type:                  "reverse_proxy",
								Enabled:               true,
								Upstream:              opts.upstream,
								LBPolicy:              opts.lbPolicy,
								HealthCheck:           opts.healthCheck,
								TransportProtocol:     opts.transportProtocol,
								TLSInsecureSkipVerify: opts.tlsInsecureSkipVerify,
							},
						},
					},
				},
			},
		},
	}
	return renderCaddyFormModel(&model)
}

func assertNoHealthDirectives(t *testing.T, out string) {
	t.Helper()
	for _, dir := range []string{"health_uri", "health_interval", "health_timeout"} {
		if strings.Contains(out, dir) {
			t.Fatalf("did not expect %s in:\n%s", dir, out)
		}
	}
}

func containsSubstring(values []string, needle string) bool {
	for _, v := range values {
		if strings.Contains(v, needle) {
			return true
		}
	}
	return false
}

func randomHealthSegment(rng *rand.Rand) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	n := 3 + rng.Intn(6)
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
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

func caddyServerRows(now time.Time, url, config string) *sqlmock.Rows {
	return caddyServerRowsWithModules(now, url, config, "{}")
}
