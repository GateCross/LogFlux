package caddy

import (
	"strings"
	"testing"
)

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

// 回源地址校验：内网/回环必须挡住，否则站点域名可被用来探测内网。
func TestIsBlockedSiteIconIP(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{host: "127.0.0.1", want: true},
		{host: "10.0.0.5", want: true},
		{host: "192.168.1.1", want: true},
		{host: "172.16.0.1", want: true},
		{host: "169.254.169.254", want: true},
		{host: "[::1]", want: true},
		{host: "8.8.8.8", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			if got := isBlockedSiteIconIP(tc.host); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
