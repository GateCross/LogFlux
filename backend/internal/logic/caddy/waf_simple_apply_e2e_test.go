package caddy

import (
	"os"
	"strings"
	"testing"

	wafmodel "logflux/model/waf"
)

// 覆盖前端「应用到 Caddy」按钮的完整链路：
// 策略 → 指令 → buildSimpleWafCandidateConfig → 最终 Caddyfile。
//
// 单独测 applyWafPolicyToCaddyConfig 不够：真实链路前面还有 ensureCorazaOrder、
// ensureWafProtectSnippet、ensureSiteImport 三步，任何一步把配置改坏都会在
// /load 时才暴露。
func TestBuildSimpleWafCandidateConfigEndToEnd(t *testing.T) {
	raw, err := os.ReadFile("testdata_live_caddyfile.txt")
	if err != nil {
		t.Skipf("缺少线上配置样本: %v", err)
	}
	config := strings.ReplaceAll(string(raw), "\r\n", "\n")

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

	// config 非空时不会读库，db 传 nil 即可。
	updated, actions, snapshot, err := buildSimpleWafCandidateConfig(nil, config, directives, simpleWafNormalizedConfig{
		Enabled:       true,
		Mode:          "on",
		Strength:      "balanced",
		SiteAddresses: []string{"pve.example.com"},
	})
	if err != nil {
		t.Fatalf("buildSimpleWafCandidateConfig() error = %v", err)
	}
	t.Logf("actions = %v", actions)

	if snapshot == nil || !snapshot.OrderReady || !snapshot.SnippetReady || !snapshot.DirectiveReady {
		t.Fatalf("候选配置 WAF 骨架不完整: %+v", snapshot)
	}
	if len(snapshot.ImportedSites) == 0 {
		t.Fatalf("候选配置没有挂载任何站点")
	}

	// 关键：三条 CRS Include 与全部策略、排除规则都必须还在。
	for _, fragment := range []string{
		"order coraza_waf first",
		"Include " + managedCRSCorazaConfigInclude,
		"Include " + managedCRSSetupConfigInclude,
		"Include " + managedCRSRulesInclude,
		"tx.blocking_paranoia_level=2",
		"tx.detection_paranoia_level=3",
		"id:1000100",
		"id:1000302",
	} {
		if !strings.Contains(updated, fragment) {
			t.Fatalf("候选配置缺少 %q", fragment)
		}
	}

	// 站点数量不能因为重渲染而变化。
	before, err := inspectWafIntegration(config)
	if err != nil {
		t.Fatalf("原配置解析失败: %v", err)
	}
	if len(snapshot.AvailableSites) != len(before.AvailableSites) {
		t.Fatalf("站点数量变化: before=%v after=%v", before.AvailableSites, snapshot.AvailableSites)
	}

	// 幂等：同一份输入再跑一次不应产生差异。
	again, _, _, err := buildSimpleWafCandidateConfig(nil, updated, directives, simpleWafNormalizedConfig{
		Enabled:       true,
		Mode:          "on",
		Strength:      "balanced",
		SiteAddresses: []string{"pve.example.com"},
	})
	if err != nil {
		t.Fatalf("二次构建失败: %v", err)
	}
	if strings.TrimSpace(again) != strings.TrimSpace(updated) {
		t.Fatalf("二次构建产生差异，说明每次应用都会让配置漂移")
	}
}
