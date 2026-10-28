package caddy

import (
	"strings"
	"testing"
)

const integrationBaseConfig = `{
  admin :2019
}

:80 {
  import geoip
}

example.com {
  reverse_proxy localhost:8080
}
`

func TestEnsureCorazaOrderAndSnippet(t *testing.T) {
	config, changed, err := ensureCorazaOrder(integrationBaseConfig)
	if err != nil {
		t.Fatalf("ensureCorazaOrder() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected order change")
	}
	if !strings.Contains(config, "order coraza_waf first") {
		t.Fatalf("expected coraza order in config")
	}

	config, changed, err = ensureWafProtectSnippet(config)
	if err != nil {
		t.Fatalf("ensureWafProtectSnippet() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected snippet change")
	}
	if !strings.Contains(config, "(waf_protect)") || !strings.Contains(config, "directives `") {
		t.Fatalf("expected waf_protect snippet with directives")
	}

	snapshot, err := inspectWafIntegration(config)
	if err != nil {
		t.Fatalf("inspectWafIntegration() error = %v", err)
	}
	if !snapshot.OrderReady || !snapshot.SnippetReady || !snapshot.DirectiveReady {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if len(snapshot.AvailableSites) != 2 {
		t.Fatalf("expected 2 sites, got %d", len(snapshot.AvailableSites))
	}
}

func TestEnsureAndRemoveSiteImport(t *testing.T) {
	config, _, err := ensureCorazaOrder(integrationBaseConfig)
	if err != nil {
		t.Fatalf("ensureCorazaOrder() error = %v", err)
	}
	config, _, err = ensureWafProtectSnippet(config)
	if err != nil {
		t.Fatalf("ensureWafProtectSnippet() error = %v", err)
	}

	config, changed, err := ensureSiteImport(config, "example.com")
	if err != nil {
		t.Fatalf("ensureSiteImport() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected import change")
	}
	if !strings.Contains(config, "example.com {\n  import waf_protect\n") {
		t.Fatalf("expected import inserted into example.com block")
	}

	config, changed, err = removeSiteImport(config, "example.com")
	if err != nil {
		t.Fatalf("removeSiteImport() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected import removal")
	}
	if strings.Contains(config, "example.com {\n  import waf_protect\n") {
		t.Fatalf("expected import removed from example.com block")
	}
}

// 观察模式站点不应被全局 apply 再补一行 waf_protect，
// 否则同一站点同时挂两个 snippet，引擎模式就没意义了。
func TestEnsureSiteImportKeepsDetectMount(t *testing.T) {
	config := `example.com {
  import waf_protect_detect
  reverse_proxy localhost:8080
}
`
	got, changed, err := ensureSiteImport(config, "example.com")
	if err != nil {
		t.Fatalf("ensureSiteImport() error = %v", err)
	}
	if changed {
		t.Fatalf("已挂观察模式时不应再插入 waf_protect，实际:\n%s", got)
	}
	if strings.Contains(got, "import waf_protect\n") {
		t.Fatalf("不应出现 waf_protect，实际:\n%s", got)
	}
	if !strings.Contains(got, "import waf_protect_detect") {
		t.Fatalf("观察模式挂载应保留，实际:\n%s", got)
	}
}

// 全局关闭 WAF 时两种挂载行都要收回。
func TestRemoveSiteImportRemovesDetectMount(t *testing.T) {
	config := `example.com {
  import waf_protect_detect
  reverse_proxy localhost:8080
}
`
	got, changed, err := removeSiteImport(config, "example.com")
	if err != nil {
		t.Fatalf("removeSiteImport() error = %v", err)
	}
	if !changed {
		t.Fatalf("应报告移除了挂载")
	}
	if strings.Contains(got, "waf_protect") {
		t.Fatalf("两种挂载都应被移除，实际:\n%s", got)
	}
	if !strings.Contains(got, "reverse_proxy localhost:8080") {
		t.Fatalf("其余指令应保留，实际:\n%s", got)
	}
}

func TestEnsureWafProtectSnippetBeforeSites(t *testing.T) {
	config := `example.com {
  import waf_protect
  reverse_proxy localhost:8080
}

(waf_protect) {
  coraza_waf {
    directives ` + "`" + `
    SecRuleEngine Off
    ` + "`" + `
  }
}
`

	updated, changed, err := ensureWafProtectSnippetBeforeSites(config)
	if err != nil {
		t.Fatalf("ensureWafProtectSnippetBeforeSites() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected snippet move")
	}
	if strings.Index(updated, "(waf_protect)") > strings.Index(updated, "import waf_protect") {
		t.Fatalf("expected waf_protect snippet before import, got:\n%s", updated)
	}
}
