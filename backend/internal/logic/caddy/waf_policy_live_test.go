package caddy

import (
	"os"
	"strings"
	"testing"

	wafmodel "logflux/model/waf"
)

// 用线上真实 Caddyfile 跑一遍应用策略，确保不会把配置改坏。
//
// 这个用例覆盖的是最容易出事的一条路径：用户点「应用到 Caddy」，
// 后端把整段 directives 重渲染后 /load。配置一旦缺了 Include 或块边界错位，
// 线上 WAF 会静默失效。
func TestApplyWafPolicyOnLiveCaddyfile(t *testing.T) {
	raw, err := os.ReadFile("testdata_live_caddyfile.txt")
	if err != nil {
		t.Skipf("缺少线上配置样本: %v", err)
	}
	config := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if strings.TrimSpace(config) == "" {
		t.Skip("线上配置样本为空")
	}

	before, err := inspectWafIntegration(config)
	if err != nil {
		t.Fatalf("解析线上配置失败: %v", err)
	}

	policy := &wafmodel.WafPolicy{
		EngineMode:                  "on",
		AuditEngine:                 "relevantonly",
		AuditLogFormat:              "json",
		AuditRelevantStatus:         wafPolicyDefaultAuditRelevantStatus,
		RequestBodyAccess:           true,
		RequestBodyLimit:            10485760,
		RequestBodyNoFilesLimit:     1048576,
		CrsTemplate:                 "balanced",
		CrsParanoiaLevel:            2,
		CrsInboundAnomalyThreshold:  5,
		CrsOutboundAnomalyThreshold: 4,
	}
	directives, err := buildWafPolicyDirectives(policy)
	if err != nil {
		t.Fatalf("buildWafPolicyDirectives() error = %v", err)
	}

	updated, err := applyWafPolicyToCaddyConfig(config, directives)
	if err != nil {
		t.Fatalf("applyWafPolicyToCaddyConfig() error = %v", err)
	}

	// 站点、snippet、全局块都不能丢。
	after, err := inspectWafIntegration(updated)
	if err != nil {
		t.Fatalf("应用后配置无法解析: %v", err)
	}
	if len(after.AvailableSites) != len(before.AvailableSites) {
		t.Fatalf("站点数量变化: before=%v after=%v", before.AvailableSites, after.AvailableSites)
	}
	if !after.OrderReady || !after.SnippetReady || !after.DirectiveReady {
		t.Fatalf("应用后 WAF 骨架缺失: %+v", after)
	}

	// 策略与排除规则必须都在，CRS Include 必须排在最后。
	for _, fragment := range []string{
		"tx.blocking_paranoia_level=2",
		"tx.detection_paranoia_level=3",
		"id:1000100",
		"id:1000302",
		"SecAuditLog /var/log/caddy/waf_audit.log",
	} {
		if !strings.Contains(updated, fragment) {
			t.Fatalf("应用后缺少 %q", fragment)
		}
	}
	for _, include := range []string{
		managedCRSCorazaConfigInclude,
		managedCRSSetupConfigInclude,
		managedCRSRulesInclude,
	} {
		if !strings.Contains(updated, "Include "+include) {
			t.Fatalf("应用后丢失 CRS Include %s", include)
		}
	}
	if strings.LastIndex(updated, managedCRSRulesInclude) < strings.LastIndex(updated, "id:1000302") {
		t.Fatalf("CRS 规则 Include 必须排在排除规则之后")
	}

	// 幂等：再应用一次不应产生差异。
	again, err := applyWafPolicyToCaddyConfig(updated, directives)
	if err != nil {
		t.Fatalf("二次应用失败: %v", err)
	}
	if again != updated {
		t.Fatalf("二次应用产生差异，说明每次保存都会让配置漂移")
	}
}
