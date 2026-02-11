package waf

import (
	"net/http"
	"net/url"
	"testing"
)

func TestValidateDomainAllowlist(t *testing.T) {
	if err := validateDomainAllowlist("api.github.com", []string{"api.github.com", "github.com"}); err != nil {
		t.Fatalf("expected host allowed, got error: %v", err)
	}

	if err := validateDomainAllowlist("codeload.github.com", []string{"github.com"}); err != nil {
		t.Fatalf("expected subdomain host allowed, got error: %v", err)
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

func TestFetchPackage_InvalidProxyURL(t *testing.T) {
	_, err := FetchPackage("https://example.com/test.zip", "/tmp/test.zip", FetchOptions{ProxyURL: "://bad-proxy"})
	if err == nil {
		t.Fatalf("expected invalid proxy url error")
	}
}

func TestFetchPackage_RejectInvalidProxyScheme(t *testing.T) {
	_, err := FetchPackage("https://example.com/test.zip", "/tmp/test.zip", FetchOptions{ProxyURL: "socks5://127.0.0.1:1080"})
	if err == nil {
		t.Fatalf("expected invalid proxy scheme error")
	}
}

func TestFetchPackage_AcceptHTTPProxyURL(t *testing.T) {
	proxyURL := "http://127.0.0.1:8080"
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if parsed.Scheme != "http" {
		t.Fatalf("unexpected proxy scheme: %s", parsed.Scheme)
	}
}
