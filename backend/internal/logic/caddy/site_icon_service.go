package caddy

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 图标抓取走服务端而不是浏览器直连：面板常以 http 暴露而站点是 https，
// 直连会被混合内容拦下，入口块里的 CSP 也只允许 img-src self/data:。
const (
	// 单个图标最多读 128KB：favicon 通常 4~32KB，留足未压缩 ico 的余量
	siteIconMaxBytes = 128 * 1024
	// 首页只读前 256KB：图标声明都在 <head> 里，读全文纯属浪费
	siteIconHomepageMaxBytes = 256 * 1024
	// 单次请求上限，以及一个站点走完 https/http 两轮探测的总预算
	siteIconRequestTimeout = 4 * time.Second
	siteIconTotalBudget    = 6 * time.Second
	// 图标几乎不变，命中后不再回源，避免每次进目录页对每个站点发一轮外网请求
	siteIconCacheTTL = 6 * time.Hour
)

// 只接受常规域名/IPv4：协议、路径、IPv6 与带分隔符的输入一律拒绝，
// 避免把任意 URL 当站点域名回源。
var siteIconHostPattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

var (
	siteIconLinkTagPattern = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	siteIconHrefPattern    = regexp.MustCompile(`(?is)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	siteIconRelPattern     = regexp.MustCompile(`(?is)\brel\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
)

var siteIconHTTPClient = &http.Client{
	Timeout: siteIconRequestTimeout,
	// 站点常见 http → https 跳转，跟两跳即可；再多说明配置异常
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return http.ErrUseLastResponse
		}
		// 跳转目标同样要过内网校验，否则内部域名 302 到 169.254.0.0/16 就能绕过入口检查
		if req.URL.Hostname() == "" || isBlockedSiteIconIP(req.URL.Hostname()) {
			return http.ErrUseLastResponse
		}
		return nil
	},
	// 连接前再校验一次解析结果，避免域名指向内网地址时被当成外网站点抓取
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if isBlockedSiteIconIP(host) {
				return nil, fmt.Errorf("回源地址不可用: %s", host)
			}
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		},
	},
}

var siteIconCache = struct {
	sync.Mutex
	items map[string]siteIconCacheItem
}{items: make(map[string]siteIconCacheItem)}

type siteIconCacheItem struct {
	icon    string
	expires time.Time
}

// resolveSiteIcon 返回站点自带图标的 data URL；抓不到返回空串，由前端回落字母头像。
func resolveSiteIcon(ctx context.Context, rawHost string) string {
	host := normalizeSiteIconHost(rawHost)
	if host == "" {
		return ""
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if icon, ok := readSiteIconCache(host); ok {
		return icon
	}

	icon := fetchSiteIcon(ctx, host)
	writeSiteIconCache(host, icon)
	return icon
}

// normalizeSiteIconHost 从站点地址里取出可回源的域名，取不到返回空串。
func normalizeSiteIconHost(rawHost string) string {
	value := strings.TrimSpace(rawHost)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return ""
		}
		value = parsed.Host
	}
	value = strings.TrimPrefix(value, "//")
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.TrimSuffix(strings.ToLower(value), ".")
	// 监听地址（:8080）与没有点的单段名不是站点域名，不做回源
	if !strings.Contains(value, ".") || !siteIconHostPattern.MatchString(value) {
		return ""
	}
	return value
}

// fetchSiteIcon 先按常规约定取 /favicon.ico，失败再解析首页的 <link rel="icon">。
// 第二步不能省：SPA 站点的图标多是带 hash 的静态文件名，根路径往往 404。
func fetchSiteIcon(ctx context.Context, host string) string {
	deadline := time.Now().Add(siteIconTotalBudget)
	if icon, reachable := probeSiteIcon(ctx, "https://"+host, deadline); icon != "" || reachable {
		return icon
	}
	// https 完全连不上（纯 http 站点）时再降级，避免无谓地重复访问一遍
	icon, _ := probeSiteIcon(ctx, "http://"+host, deadline)
	return icon
}

// probeSiteIcon 探测一个协议下的图标；reachable 表示站点本身是否可访问。
// 三次请求共享同一个总预算，站点慢时不会把整个请求拖成超时。
func probeSiteIcon(ctx context.Context, baseURL string, deadline time.Time) (string, bool) {
	timeoutCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	if icon := requestSiteIcon(timeoutCtx, baseURL+"/favicon.ico"); icon != "" {
		return icon, true
	}

	page, ok := requestSiteText(timeoutCtx, baseURL+"/", siteIconHomepageMaxBytes)
	if !ok {
		return "", false
	}
	href := iconHrefFromHTML(page, baseURL+"/")
	if href == "" {
		return "", true
	}
	return requestSiteIcon(timeoutCtx, href), true
}

// requestSiteIcon 取单个 URL 并转成 data URL；非图片响应按没取到处理。
func requestSiteIcon(ctx context.Context, target string) string {
	resp, err := doSiteIconRequest(ctx, target)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	body, ok := readLimitedBody(resp, siteIconMaxBytes)
	if !ok {
		return ""
	}
	return siteIconDataURL(resp.Header.Get("Content-Type"), body)
}

func requestSiteText(ctx context.Context, target string, limit int64) (string, bool) {
	resp, err := doSiteIconRequest(ctx, target)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false
	}
	body, ok := readLimitedBody(resp, limit)
	if !ok {
		return "", false
	}
	return string(body), true
}

func doSiteIconRequest(ctx context.Context, target string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	// 部分站点按 UA 决定是否返回图标，伪装成常见浏览器最省事
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LogFlux site icon fetch)")
	req.Header.Set("Accept", "image/*,text/html;q=0.8")
	return siteIconHTTPClient.Do(req)
}

// readLimitedBody 读满上限即视为超限失败，半截图片没有展示价值。
func readLimitedBody(resp *http.Response, limit int64) ([]byte, bool) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || int64(len(body)) > limit || len(body) == 0 {
		return nil, false
	}
	return body, true
}

// siteIconDataURL 把响应体转成 data URL；声明与字节都不像图片时返回空串。
func siteIconDataURL(contentType string, body []byte) string {
	mime := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if mime == "image/vnd.microsoft.icon" {
		mime = "image/x-icon"
	}
	if !strings.HasPrefix(mime, "image/") {
		// 不少服务器把 .ico 当 application/octet-stream 返回，只能按字节兜底
		mime = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(body), ";")[0]))
		if !strings.HasPrefix(mime, "image/") {
			return ""
		}
	}
	// SVG 常被错误地声明为 image/svg+xml 的其实是 HTML 错误页，认不出根标签就不收
	if mime == "image/svg+xml" && !bytes.Contains(bytes.ToLower(body), []byte("<svg")) {
		return ""
	}
	return fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(body))
}

// iconHrefFromHTML 从首页 HTML 里找出图标声明并解析成绝对 URL。
func iconHrefFromHTML(page, baseURL string) string {
	for _, tag := range siteIconLinkTagPattern.FindAllString(page, -1) {
		rel := strings.ToLower(strings.TrimSpace(firstRegexGroup(siteIconRelPattern, tag)))
		if !strings.Contains(rel, "icon") {
			continue
		}
		href := strings.TrimSpace(firstRegexGroup(siteIconHrefPattern, tag))
		if href == "" {
			continue
		}
		// 内联 data URL 本身就是成品，直接透传
		if strings.HasPrefix(strings.ToLower(href), "data:image/") {
			return href
		}
		base, err := url.Parse(baseURL)
		if err != nil {
			continue
		}
		ref, err := url.Parse(href)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(ref)
		if resolved.Scheme != "http" && resolved.Scheme != "https" {
			continue
		}
		return resolved.String()
	}
	return ""
}

// firstRegexGroup 取三个分支（双引号、单引号、无引号）里第一个命中的分组。
func firstRegexGroup(pattern *regexp.Regexp, s string) string {
	match := pattern.FindStringSubmatch(s)
	if match == nil {
		return ""
	}
	for _, group := range match[1:] {
		if group != "" {
			return group
		}
	}
	return ""
}

// isBlockedSiteIconIP 拒绝回环、内网与链路本地地址。
// 图标抓取是「按用户给的域名出网」的入口，必须挡住指向内网的解析结果，
// 否则一个指向 127.0.0.1 的域名就能把面板变成内网探针。
func isBlockedSiteIconIP(host string) bool {
	trimmed := strings.Trim(host, "[]")
	if ip := net.ParseIP(trimmed); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
	}
	// 主机名要解析后才能判断；解析失败交给拨号阶段报错
	ips, err := net.LookupIP(trimmed)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if isBlockedSiteIconIP(ip.String()) {
			return true
		}
	}
	return false
}

func readSiteIconCache(host string) (string, bool) {	siteIconCache.Lock()
	defer siteIconCache.Unlock()
	item, ok := siteIconCache.items[host]
	if !ok || time.Now().After(item.expires) {
		return "", false
	}
	return item.icon, true
}

func writeSiteIconCache(host, icon string) {
	siteIconCache.Lock()
	defer siteIconCache.Unlock()
	// 失效项按需清掉，避免地图无限增长
	now := time.Now()
	for key, item := range siteIconCache.items {
		if now.After(item.expires) {
			delete(siteIconCache.items, key)
		}
	}
	siteIconCache.items[host] = siteIconCacheItem{icon: icon, expires: now.Add(siteIconCacheTTL)}
}
