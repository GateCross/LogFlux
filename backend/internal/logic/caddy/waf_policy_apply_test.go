package caddy

import (
	wafmodel "logflux/model/waf"
	"strings"
	"testing"
)

// 应用策略时必须保留 CRS 的三条 Include。
//
// 曾经的做法是把 directives 反引号块整段换成策略文本，块里的 Include 一并消失：
// load_owasp_crs 仍为 true，但一条 CRS 规则都没加载，WAF 变成只跑排除规则的空壳。
func TestApplyWafPolicyKeepsCRSIncludes(t *testing.T) {
	config := renderWafProtectSnippet("\n") + "\nexample.com {\n  import waf_protect\n  reverse_proxy localhost:8080\n}\n"

	directives, err := buildWafPolicyDirectives(testWafPolicy())
	if err != nil {
		t.Fatalf("buildWafPolicyDirectives() error = %v", err)
	}

	updated, err := applyWafPolicyToCaddyConfig(config, directives)
	if err != nil {
		t.Fatalf("applyWafPolicyToCaddyConfig() error = %v", err)
	}

	for _, include := range []string{
		managedCRSCorazaConfigInclude,
		managedCRSSetupConfigInclude,
		managedCRSRulesInclude,
	} {
		if !strings.Contains(updated, "Include "+include) {
			t.Fatalf("应用策略后丢失 %s，输出=\n%s", include, updated)
		}
	}

	// 规则加载必须排在策略与排除规则之后，否则排除规则不生效。
	if strings.Index(updated, managedCRSRulesInclude) < strings.Index(updated, "id:900000") {
		t.Fatalf("CRS 规则 Include 必须位于策略之后，输出=\n%s", updated)
	}
	if strings.Index(updated, managedCRSRulesInclude) < strings.Index(updated, "id:1000302") {
		t.Fatalf("CRS 规则 Include 必须位于排除规则之后，输出=\n%s", updated)
	}
}

// 连续应用两次结果必须一致，否则每次保存都会让配置漂移。
func TestApplyWafPolicyIdempotent(t *testing.T) {
	config := renderWafProtectSnippet("\n") + "\nexample.com {\n  import waf_protect\n}\n"

	directives, err := buildWafPolicyDirectives(testWafPolicy())
	if err != nil {
		t.Fatalf("buildWafPolicyDirectives() error = %v", err)
	}

	first, err := applyWafPolicyToCaddyConfig(config, directives)
	if err != nil {
		t.Fatalf("first apply error = %v", err)
	}
	second, err := applyWafPolicyToCaddyConfig(first, directives)
	if err != nil {
		t.Fatalf("second apply error = %v", err)
	}
	if first != second {
		t.Fatalf("应用两次结果不一致，first=\n%s\nsecond=\n%s", first, second)
	}
}

// 多个 directives 块都要被更新，不能只改第一个。
func TestApplyWafPolicyUpdatesAllBlocks(t *testing.T) {
	config := "{\n  order coraza_waf first\n}\n\n" +
		"(waf_a) {\n  coraza_waf {\n    directives `\n    Include @owasp_crs/*.conf\n    `\n  }\n}\n\n" +
		"(waf_b) {\n  coraza_waf {\n    directives `\n    Include @owasp_crs/*.conf\n    `\n  }\n}\n"

	directives, err := buildWafPolicyDirectives(testWafPolicy())
	if err != nil {
		t.Fatalf("buildWafPolicyDirectives() error = %v", err)
	}

	updated, err := applyWafPolicyToCaddyConfig(config, directives)
	if err != nil {
		t.Fatalf("applyWafPolicyToCaddyConfig() error = %v", err)
	}
	if count := strings.Count(updated, "id:900000"); count != 2 {
		t.Fatalf("期望两个 directives 块都被更新，实际命中 %d 次", count)
	}
}

// 站点级引擎模式：拦截 snippet 写策略模式，观察 snippet 强制 DetectionOnly。
//
// 这是"防护模式只作用于单站点"的核心保证。曾经所有站点共用一份 waf_protect，
// SecRuleEngine 只有一处，切换一个站点等于切换全部站点。
func TestApplyWafPolicyRespectsPerSiteEngineMode(t *testing.T) {
	config := renderWafProtectSnippet("\n") + "\n" +
		renderWafDetectSnippet("\n") + "\n" +
		"pve.example.com {\n  import waf_protect\n}\n\n" +
		"blog.example.com {\n  import waf_protect_detect\n}\n"

	policy := testWafPolicy()
	policy.EngineMode = "on"
	directives, err := buildWafPolicyDirectives(policy)
	if err != nil {
		t.Fatalf("buildWafPolicyDirectives() error = %v", err)
	}

	updated, err := applyWafPolicyToCaddyConfig(config, directives)
	if err != nil {
		t.Fatalf("applyWafPolicyToCaddyConfig() error = %v", err)
	}

	protectBody := snippetBody(t, updated, "waf_protect")
	detectBody := snippetBody(t, updated, "waf_protect_detect")

	if !strings.Contains(protectBody, "SecRuleEngine On") {
		t.Fatalf("拦截 snippet 应为 On，实际=\n%s", protectBody)
	}
	if strings.Contains(protectBody, "SecRuleEngine DetectionOnly") {
		t.Fatalf("拦截 snippet 不应被降级为 DetectionOnly，实际=\n%s", protectBody)
	}
	if !strings.Contains(detectBody, "SecRuleEngine DetectionOnly") {
		t.Fatalf("观察 snippet 必须固定 DetectionOnly，实际=\n%s", detectBody)
	}
	if strings.Contains(detectBody, "SecRuleEngine On") {
		t.Fatalf("观察 snippet 不应被执行模式覆盖，实际=\n%s", detectBody)
	}
}

// 观察模式 snippet 即使策略是 Off，也必须保持 DetectionOnly：
// 站点选了这个挂载方式，就不该被全局开关改掉检测行为。
func TestApplyWafPolicyKeepsDetectSnippetWhenPolicyOff(t *testing.T) {
	config := renderWafDetectSnippet("\n") + "\nblog.example.com {\n  import waf_protect_detect\n}\n"

	policy := testWafPolicy()
	policy.EngineMode = "off"
	directives, err := buildWafPolicyDirectives(policy)
	if err != nil {
		t.Fatalf("buildWafPolicyDirectives() error = %v", err)
	}

	updated, err := applyWafPolicyToCaddyConfig(config, directives)
	if err != nil {
		t.Fatalf("applyWafPolicyToCaddyConfig() error = %v", err)
	}
	if body := snippetBody(t, updated, "waf_protect_detect"); !strings.Contains(body, "SecRuleEngine DetectionOnly") {
		t.Fatalf("观察 snippet 不应变成 Off，实际=\n%s", body)
	}
}

// snippetBody 取出指定 snippet 的块文本，避免跨块误判。
func snippetBody(t *testing.T, config, name string) string {
	t.Helper()
	blocks, lines, err := parseTopLevelCaddyBlocks(config)
	if err != nil {
		t.Fatalf("parseTopLevelCaddyBlocks() error = %v", err)
	}
	for _, block := range blocks {
		if block.Kind == "snippet" && block.Name == name {
			return joinBlockLines(lines, block)
		}
	}
	t.Fatalf("未找到 snippet %s，配置=\n%s", name, config)
	return ""
}

func testWafPolicy() *wafmodel.WafPolicy {
	return &wafmodel.WafPolicy{
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
}
