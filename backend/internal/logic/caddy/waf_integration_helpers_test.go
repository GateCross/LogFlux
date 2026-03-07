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
