package route

import (
	"testing"
)

func TestMenuPermissionKey_LogPages(t *testing.T) {
	if got := menuPermissionKey("caddy_system_log"); got != "logs" {
		t.Fatalf("expected caddy_system_log -> logs, got %q", got)
	}
	if got := menuPermissionKey("caddy_system-log"); got != "logs" {
		t.Fatalf("expected caddy_system-log -> logs, got %q", got)
	}
	if got := menuPermissionKey("caddy_log"); got != "logs_caddy" {
		t.Fatalf("expected caddy_log -> logs_caddy, got %q", got)
	}
	if got := menuPermissionKey("security"); got != "security" {
		t.Fatalf("expected security -> security, got %q", got)
	}
}
