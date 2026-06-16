package caddy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	caddymodel "logflux/model/caddy"
)

func TestPostCaddyTextTreatsHTTP200BodyErrorAsFailure(t *testing.T) {
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/load" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"file":"Caddyfile","line":2,"message":"format warning"}]{"error":"loading config: invalid WAF config"}`))
	}))
	defer caddyMock.Close()

	status, body, err := postCaddyText(
		&caddymodel.CaddyServer{Url: caddyMock.URL},
		"/load",
		"text/caddyfile",
		":80 { respond ok }",
	)
	if err == nil {
		t.Fatalf("expected body error to fail")
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if !strings.Contains(string(body), "invalid WAF config") {
		t.Fatalf("expected raw body to be preserved, got %s", string(body))
	}
	if !strings.Contains(err.Error(), "invalid WAF config") {
		t.Fatalf("expected Caddy error message, got %v", err)
	}
}

func TestPostCaddyTextAllowsHTTP200Warnings(t *testing.T) {
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/load" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"file":"Caddyfile","line":2,"message":"format warning"}]`))
	}))
	defer caddyMock.Close()

	_, _, err := postCaddyText(
		&caddymodel.CaddyServer{Url: caddyMock.URL},
		"/load",
		"text/caddyfile",
		":80 { respond ok }",
	)
	if err != nil {
		t.Fatalf("expected warnings-only response to succeed, got %v", err)
	}
}
