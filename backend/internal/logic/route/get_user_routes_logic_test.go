package route

import (
	"testing"
)

func TestMenuPermissionKey_LogPages(t *testing.T) {
	if got := menuPermissionKey("caddy"); got != "caddy" {
		t.Fatalf("expected caddy -> caddy, got %q", got)
	}
	if got := menuPermissionKey("caddy_config"); got != "caddy_config" {
		t.Fatalf("expected caddy_config -> caddy_config, got %q", got)
	}
	if got := menuPermissionKey("caddy_access"); got != "caddy_access" {
		t.Fatalf("expected caddy_access -> caddy_access, got %q", got)
	}
	if got := menuPermissionKey("caddy_source"); got != "caddy_source" {
		t.Fatalf("expected caddy_source -> caddy_source, got %q", got)
	}
	if got := menuPermissionKey("caddy_system_log"); got != "logs" {
		t.Fatalf("expected caddy_system_log -> logs, got %q", got)
	}
	if got := menuPermissionKey("caddy_system-log"); got != "logs" {
		t.Fatalf("expected caddy_system-log -> logs, got %q", got)
	}
	if got := menuPermissionKey("caddy_log"); got != "logs_caddy" {
		t.Fatalf("expected caddy_log -> logs_caddy, got %q", got)
	}
	if got := menuPermissionKey("cron"); got != "cron" {
		t.Fatalf("expected cron -> cron, got %q", got)
	}
}
