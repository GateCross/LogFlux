package caddy

import (
	"strings"
	"testing"
)

// 基准配置用真实 Caddyfile 形态：trusted_proxies 嵌在 servers 子块里。
const mergeTestBaseConfig = `{
  admin localhost:2019
  order coraza_waf first
  auto_https disable_redirects
  servers {
    trusted_proxies static 10.0.0.0/8
    timeouts {
      read_body 10s
    }
  }
}

:80 {
  import waf_protect
}
`

func TestMergeGlobalOptionsReplacesConflicts(t *testing.T) {
	debugOn := true
	opts := CaddyGlobalOptionsValue{
		AutoHttps:      "off",
		TrustedProxies: []string{"192.168.0.0/16", "10.0.0.0/8"},
		Debug:          &debugOn,
	}
	got := mergeGlobalOptions(mergeTestBaseConfig, opts)
	if !strings.Contains(got, "auto_https off") {
		t.Fatalf("应写入 auto_https off:\n%s", got)
	}
	if !strings.Contains(got, "trusted_proxies static 192.168.0.0/16 10.0.0.0/8") {
		t.Fatalf("应替换 servers 块内 trusted_proxies:\n%s", got)
	}
	if !strings.Contains(got, "debug") {
		t.Fatalf("应写入 debug:\n%s", got)
	}
	if strings.Contains(got, "disable_redirects") {
		t.Fatalf("旧 auto_https 值应被删除:\n%s", got)
	}
	// trusted_proxies 不得以顶层行出现（非法 Caddyfile）：顶层行缩进恰为 2 空格。
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "  trusted_proxies") {
			t.Fatalf("trusted_proxies 不得出现在全局块顶层:\n%s", got)
		}
	}
	if strings.Count(got, "servers {") != 1 {
		t.Fatalf("servers 子块应恰好一个:\n%s", got)
	}
	if !strings.Contains(got, "admin localhost:2019") || !strings.Contains(got, "order coraza_waf first") {
		t.Fatalf("未接管的行不应变化:\n%s", got)
	}
	if !strings.Contains(got, "read_body 10s") {
		t.Fatalf("servers 块内其他子指令不应受影响:\n%s", got)
	}
	if !strings.Contains(got, "import waf_protect") {
		t.Fatalf("站点块不应受影响:\n%s", got)
	}
}

func TestMergeGlobalOptionsIdempotent(t *testing.T) {
	debugTrue := true
	opts := CaddyGlobalOptionsValue{AutoHttps: "disable_redirects", TrustedProxies: []string{"10.0.0.0/8"}, Debug: &debugTrue}
	once := mergeGlobalOptions(mergeTestBaseConfig, opts)
	twice := mergeGlobalOptions(once, opts)
	if once != twice {
		t.Fatalf("合并应幂等\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestMergeGlobalOptionsEmptyKeepsExisting(t *testing.T) {
	got := mergeGlobalOptions(mergeTestBaseConfig, CaddyGlobalOptionsValue{})
	if got != mergeTestBaseConfig {
		t.Fatalf("空选项不应改动配置:\n%s", got)
	}
}

func TestMergeGlobalOptionsCreatesBlockWhenMissing(t *testing.T) {
	config := ":80 {\n  respond \"hi\"\n}\n"
	debugOn := true
	opts := CaddyGlobalOptionsValue{Debug: &debugOn, TrustedProxies: []string{"10.0.0.0/8"}}
	got := mergeGlobalOptions(config, opts)
	if !strings.HasPrefix(strings.TrimSpace(got), "{\n  debug\n  servers {\n    trusted_proxies static 10.0.0.0/8\n  }\n}") {
		t.Fatalf("无全局块时应在头部生成（含 servers 子块）:\n%s", got)
	}
	if !strings.Contains(got, "respond") {
		t.Fatalf("原内容应保留:\n%s", got)
	}
}

func TestMergeGlobalOptionsAppendsServersBlockWhenMissing(t *testing.T) {
	config := "{\n  admin localhost:2019\n}\n"
	opts := CaddyGlobalOptionsValue{TrustedProxies: []string{"172.16.0.0/12"}}
	got := mergeGlobalOptions(config, opts)
	if !strings.Contains(got, "servers {\n    trusted_proxies static 172.16.0.0/12\n  }") {
		t.Fatalf("无 servers 块时应追加子块:\n%s", got)
	}
}

func TestMergeGlobalOptionsStripsLegacyTopLevelTrustedProxies(t *testing.T) {
	config := "{\n  trusted_proxies static 10.0.0.0/8\n}\n"
	opts := CaddyGlobalOptionsValue{TrustedProxies: []string{"192.168.0.0/16"}}
	got := mergeGlobalOptions(config, opts)
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "  trusted_proxies") {
			t.Fatalf("历史顶层写法应被清除:\n%s", got)
		}
	}
	if !strings.Contains(got, "servers {") {
		t.Fatalf("应生成 servers 子块:\n%s", got)
	}
}

func TestMergeGlobalOptionsCRLF(t *testing.T) {
	config := strings.ReplaceAll(mergeTestBaseConfig, "\n", "\r\n")
	debugOn := true
	opts := CaddyGlobalOptionsValue{Debug: &debugOn, TrustedProxies: []string{"10.0.0.0/8"}}
	got := mergeGlobalOptions(config, opts)
	if !strings.Contains(got, "\r\n  debug\r\n") {
		t.Fatalf("CRLF 应保留:\n%q", got)
	}
	if strings.Contains(got, "\n") && strings.Count(got, "\r\n") != strings.Count(got, "\n") {
		t.Fatalf("换行风格应统一:\n%q", got)
	}
}
