package caddy

import (
	"strings"
	"testing"
)

// 回源目标选取：上游地址优先且保留端口与协议，无上游时才落到站点域名。
func TestParseSiteIconTarget(t *testing.T) {
	cases := []struct {
		name     string
		upstream string
		host     string
		want     string // key() 结果，nil 目标为空串
	}{
		{name: "上游内网 IP 带端口", upstream: "192.168.50.100:8006", host: "example.top", want: "//192.168.50.100:8006"},
		{name: "上游显式 HTTPS", upstream: "https://192.168.50.100:8006", host: "example.top", want: "https://192.168.50.100:8006"},
		{name: "上游带路径剥掉路径", upstream: "http://localhost:8080/prefix", host: "test.local", want: "http://localhost:8080"},
		{name: "多上游取第一个", upstream: "10.0.0.1:8080 10.0.0.2:8080", want: "//10.0.0.1:8080"},
		{name: "非 http 上游退回域名", upstream: "unix//run/app.sock", host: "example.com", want: "//example.com"},
		{name: "无上游走域名", host: "example.com", want: "//example.com"},
		{name: "空输入", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := parseSiteIconTarget(tc.upstream, tc.host)
			got := ""
			if target != nil {
				got = target.key()
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// 站点域名校验：只有能安全回源的 host 才允许发请求。
func TestNormalizeSiteIconHost(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "纯域名", in: "example.com", want: "example.com"},
		{name: "带端口", in: "example.com:8443", want: "example.com"},
		{name: "带大写与尾点", in: "Example.COM.", want: "example.com"},
		{name: "完整 URL", in: "https://www.example.com/path", want: "www.example.com"},
		{name: "监听地址", in: ":80", want: ""},
		{name: "单段主机名", in: "localhost", want: ""},
		{name: "空值", in: "  ", want: ""},
		{name: "带路径的非 URL", in: "example.com/a", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeSiteIconHost(tc.in); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// 上游主机校验：内网 IP、localhost 与容器名都要放行，分隔符输入要挡住。
func TestIsValidSiteIconHost(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{in: "192.168.50.100:8006", want: true},
		{in: "127.0.0.1:8080", want: true},
		{in: "localhost:8080", want: true},
		{in: "web-app:3000", want: true},
		{in: "[::1]:8080", want: true},
		{in: "example.com", want: true},
		{in: "  ", want: false},
		{in: "invalid/path", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := isValidSiteIconHost(tc.in); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// 首页图标声明解析：相对地址要按首页 URL 补全，非图片 scheme 一律丢弃。
func TestIconHrefFromHTML(t *testing.T) {
	cases := []struct {
		name string
		page string
		want string
	}{
		{
			name: "相对路径",
			page: `<html><head><link rel="icon" href="/static/logo.png"></head></html>`,
			want: "https://example.com/static/logo.png",
		},
		{
			name: "单引号与绝对地址",
			page: `<link rel='shortcut icon' href='https://cdn.example.net/a.ico'>`,
			want: "https://cdn.example.net/a.ico",
		},
		{
			name: "内联 data URL 原样返回",
			page: `<link rel="icon" href="data:image/png;base64,AAA">`,
			want: "data:image/png;base64,AAA",
		},
		{
			name: "优先 icon 而非其他 link",
			page: `<link rel="stylesheet" href="/a.css"><link rel="apple-touch-icon" href="/b.png">`,
			want: "https://example.com/b.png",
		},
		{
			name: "没有图标声明",
			page: `<html><head><title>x</title></head></html>`,
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := iconHrefFromHTML(tc.page, "https://example.com/")
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// 字节兜底：服务器把 ico 声明成 octet-stream 时仍要认成图片。
func TestSiteIconDataURL(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n")
	if got := siteIconDataURL("application/octet-stream", png); !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("expected png data url, got %q", got)
	}
	// HTML 错误页被声明成 svg 时不能当图标返回
	if got := siteIconDataURL("image/svg+xml", []byte("<html>404</html>")); got != "" {
		t.Fatalf("expected empty for fake svg, got %q", got)
	}
	if got := siteIconDataURL("text/html", []byte("<html>404</html>")); got != "" {
		t.Fatalf("expected empty for html, got %q", got)
	}
}
