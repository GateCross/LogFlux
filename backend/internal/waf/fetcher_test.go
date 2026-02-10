package waf

import (
	"net/http"
	"testing"
)

func TestValidateDomainAllowlist(t *testing.T) {
	if err := validateDomainAllowlist("api.github.com", []string{"api.github.com", "github.com"}); err != nil {
		t.Fatalf("expected host allowed, got error: %v", err)
	}

	if err := validateDomainAllowlist("malicious.example", []string{"api.github.com"}); err == nil {
		t.Fatalf("expected host blocked")
	}
}

func TestApplyAuth(t *testing.T) {
	reqToken, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	applyAuth(reqToken, "token", "abc123")
	if reqToken.Header.Get("Authorization") != "Bearer abc123" {
		t.Fatalf("unexpected token auth header: %s", reqToken.Header.Get("Authorization"))
	}

	reqBasic, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	applyAuth(reqBasic, "basic", "user:pass")
	username, password, ok := reqBasic.BasicAuth()
	if !ok || username != "user" || password != "pass" {
		t.Fatalf("unexpected basic auth: ok=%v user=%s pass=%s", ok, username, password)
	}
}

func TestFetchPackage_RejectNonHTTPS(t *testing.T) {
	_, err := FetchPackage("http://example.com/test.zip", "/tmp/test.zip", FetchOptions{})
	if err == nil {
		t.Fatalf("expected non-https error")
	}
}
