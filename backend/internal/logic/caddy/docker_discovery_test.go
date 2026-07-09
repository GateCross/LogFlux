package caddy

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Docker 标签发现 → 仅会话候选（无 DB、无 /load）。
// **验证：Requirements 1.2（不自动 Apply_Path）** 及 Phase 2 Docker 发现设计。

func TestParseDockerDiscoveryCandidates_EnabledWithHost(t *testing.T) {
	containers := []DockerContainerSummary{
		{
			ID:     "abcdef0123456789",
			Name:   "/demo-api",
			Status: "Up 2 hours",
			Labels: map[string]string{
				"logflux.enable":          "true",
				"logflux.host":            "api.example.com",
				"logflux.port":            "8080",
				"logflux.name":            "Demo API",
				"logflux.tls":             "internal",
				"logflux.lb_policy":       "least_conn",
				"logflux.health.path":     "/healthz",
				"logflux.health.interval": "10s",
				"logflux.health.timeout":  "3s",
			},
			PrivatePorts: []int{8080},
		},
	}

	list := ParseDockerDiscoveryCandidates(containers)
	if len(list) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(list))
	}
	c := list[0]
	if !c.Valid {
		t.Fatalf("expected valid candidate, reason=%s", c.Reason)
	}
	if c.Name != "Demo API" {
		t.Fatalf("name: got %q", c.Name)
	}
	if len(c.Domains) != 1 || c.Domains[0] != "api.example.com" {
		t.Fatalf("domains: %+v", c.Domains)
	}
	if c.Upstream != "demo-api:8080" {
		t.Fatalf("upstream: got %q want demo-api:8080", c.Upstream)
	}
	if c.TlsMode != "internal" {
		t.Fatalf("tls: %q", c.TlsMode)
	}
	if c.LbPolicy != "least_conn" {
		t.Fatalf("lb: %q", c.LbPolicy)
	}
	if c.HealthPath != "/healthz" || c.HealthInterval != "10s" || c.HealthTimeout != "3s" {
		t.Fatalf("health: path=%s interval=%s timeout=%s", c.HealthPath, c.HealthInterval, c.HealthTimeout)
	}
	if !strings.HasPrefix(c.CandidateId, "docker-") {
		t.Fatalf("candidateId prefix: %s", c.CandidateId)
	}
}

func TestParseDockerDiscoveryCandidates_IgnoresWithoutEnable(t *testing.T) {
	containers := []DockerContainerSummary{
		{
			ID:   "111",
			Name: "/no-label",
			Labels: map[string]string{
				"logflux.host": "x.example.com",
			},
		},
		{
			ID:   "222",
			Name: "/disabled",
			Labels: map[string]string{
				"logflux.enable": "false",
				"logflux.host":   "y.example.com",
			},
		},
	}
	list := ParseDockerDiscoveryCandidates(containers)
	if len(list) != 0 {
		t.Fatalf("expected 0, got %d", len(list))
	}
}

func TestParseDockerDiscoveryCandidates_MissingHostInvalid(t *testing.T) {
	list := ParseDockerDiscoveryCandidates([]DockerContainerSummary{
		{
			ID:   "abc123",
			Name: "/svc",
			Labels: map[string]string{
				"logflux.enable": "1",
				"logflux.port":   "9000",
			},
			PrivatePorts: []int{9000},
		},
	})
	if len(list) != 1 || list[0].Valid {
		t.Fatalf("expected one invalid candidate, got %+v", list)
	}
	if !strings.Contains(list[0].Reason, "host") {
		t.Fatalf("chinese reason expected, got %q", list[0].Reason)
	}
}

func TestParseDockerDiscoveryCandidates_HealthPathMustStartWithSlash(t *testing.T) {
	list := ParseDockerDiscoveryCandidates([]DockerContainerSummary{
		{
			ID:   "abc123",
			Name: "/svc",
			Labels: map[string]string{
				"logflux.enable":      "yes",
				"logflux.host":        "svc.example.com",
				"logflux.port":        "80",
				"logflux.health.path": "health",
			},
		},
	})
	if len(list) != 1 || list[0].Valid {
		t.Fatalf("expected invalid: %+v", list)
	}
	if !strings.Contains(list[0].Reason, "/") {
		t.Fatalf("reason: %q", list[0].Reason)
	}
}

func TestParseDockerDiscoveryCandidates_MultiHostAndUpstreamOverride(t *testing.T) {
	list := ParseDockerDiscoveryCandidates([]DockerContainerSummary{
		{
			ID:   "deadbeefcafe",
			Name: "/web",
			Labels: map[string]string{
				"logflux.enable":   "true",
				"logflux.host":     "a.example.com, b.example.com",
				"logflux.upstream": "10.0.0.5:3000",
			},
		},
	})
	if len(list) != 1 || !list[0].Valid {
		t.Fatalf("unexpected: %+v", list)
	}
	if len(list[0].Domains) != 2 {
		t.Fatalf("domains: %+v", list[0].Domains)
	}
	if list[0].Upstream != "10.0.0.5:3000" {
		t.Fatalf("upstream override: %q", list[0].Upstream)
	}
}

func TestParseDockerDiscoveryCandidates_UsesNetworkIPWhenNoName(t *testing.T) {
	list := ParseDockerDiscoveryCandidates([]DockerContainerSummary{
		{
			ID:   "idonly",
			Name: "",
			Labels: map[string]string{
				"logflux.enable": "true",
				"logflux.host":   "ip.example.com",
				"logflux.port":   "8080",
			},
			NetworkIPs: map[string]string{
				"bridge": "172.18.0.12",
			},
		},
	})
	if len(list) != 1 || !list[0].Valid {
		t.Fatalf("unexpected: %+v", list)
	}
	if list[0].Upstream != "172.18.0.12:8080" {
		t.Fatalf("upstream: %q", list[0].Upstream)
	}
}

func TestDiscoverDockerServicesLogic_NoLoadAndNoDB(t *testing.T) {
	// 类属性：发现路径不改配置；列表函数纯函数；空列表合法。
	logic := NewDiscoverDockerServicesLogic(context.Background(), nil)
	var loadLikeCalls int
	logic.listFn = func(ctx context.Context) ([]DockerContainerSummary, error) {
		// 模拟容器；设计上断言无 /load 副作用
		if strings.Contains(fmt.Sprint(ctx), "/load") {
			loadLikeCalls++
		}
		return []DockerContainerSummary{
			{
				ID:   "aaa111",
				Name: "/ok",
				Labels: map[string]string{
					"logflux.enable": "true",
					"logflux.host":   "ok.example.com",
					"logflux.port":   "80",
				},
			},
		}, nil
	}

	resp, err := logic.DiscoverDockerServices()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp == nil || len(resp.List) != 1 {
		t.Fatalf("resp: %+v", resp)
	}
	if !resp.List[0].Valid {
		t.Fatalf("expected valid: %+v", resp.List[0])
	}
	if resp.ScannedAt == "" {
		t.Fatalf("scannedAt required")
	}
	if loadLikeCalls != 0 {
		t.Fatalf("must never target /load")
	}
	if !strings.Contains(resp.Message, "未调用 /load") {
		t.Fatalf("message should state no /load: %s", resp.Message)
	}
}

func TestDiscoverDockerServicesLogic_DockerUnavailableSoftFail(t *testing.T) {
	logic := NewDiscoverDockerServicesLogic(context.Background(), nil)
	logic.listFn = func(ctx context.Context) ([]DockerContainerSummary, error) {
		return nil, fmt.Errorf("Docker socket 不可用: /var/run/docker.sock")
	}
	resp, err := logic.DiscoverDockerServices()
	if err != nil {
		t.Fatalf("soft-fail must not return error: %v", err)
	}
	if resp == nil {
		t.Fatal("nil resp")
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list, got %d", len(resp.List))
	}
	if resp.Message == "" {
		t.Fatal("expected chinese/error message")
	}
}

func TestDiscoverDockerServicesLogic_EmptyContainers(t *testing.T) {
	logic := NewDiscoverDockerServicesLogic(context.Background(), nil)
	logic.listFn = func(ctx context.Context) ([]DockerContainerSummary, error) {
		return []DockerContainerSummary{}, nil
	}
	resp, err := logic.DiscoverDockerServices()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.List == nil {
		t.Fatal("list must be non-nil empty")
	}
	if len(resp.List) != 0 {
		t.Fatalf("want 0 got %d", len(resp.List))
	}
}

func TestTruthyAndNormalizeHelpers(t *testing.T) {
	if !isTruthyLabel("YES") || isTruthyLabel("no") {
		t.Fatal("truthy")
	}
	if normalizeTLSMode("http") != "off" || normalizeTLSMode("") != "auto" {
		t.Fatal("tls")
	}
	if normalizeLBPolicy("ip-hash") != "ip_hash" {
		t.Fatal("lb")
	}
	if len(splitCSV("a, b;c")) != 3 {
		t.Fatal("csv")
	}
}
