package caddy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	caddymodel "logflux/model/caddy"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 图标抓取走服务端而不是浏览器直连：面板常以 http 暴露而站点是 https，
// 直连会被混合内容拦下，入口块里的 CSP 也只允许 img-src self/data:。
// 回源目标优先用上游地址：域名可能没解析、被防火墙挡住或回流不通，上游才是真跑着的服务。
const (
	// 单个图标最多读 128KB：favicon 通常 4~32KB，留足未压缩 ico 的余量
	siteIconMaxBytes = 128 * 1024
	// 首页只读前 256KB：图标声明都在 <head> 里，读全文纯属浪费
	siteIconHomepageMaxBytes = 256 * 1024
	// 单次请求上限，以及一个目标走完 http/https 两轮探测的总预算
	siteIconRequestTimeout = 4 * time.Second
	siteIconTotalBudget    = 6 * time.Second
	// 图标几乎不变，命中后不再回源，避免每次进目录页对每个站点发一轮请求
	siteIconCacheTTL = 6 * time.Hour
)

var (
	siteIconLinkTagPattern = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	siteIconHrefPattern    = regexp.MustCompile(`(?is)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	siteIconRelPattern     = regexp.MustCompile(`(?is)\brel\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	siteIconHostPattern    = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)
)

var siteIconHTTPClient = &http.Client{
	Timeout: siteIconRequestTimeout,
	// 站点常见 http → https 跳转，跟两跳即可；再多说明配置异常
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return http.ErrUseLastResponse
		}
		return nil
	},
	Transport: &http.Transport{
		// 上游多是内网面板（PVE、各类管理台），自签证书是常态，只取 favicon 不涉敏感数据
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 上游自签证书
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

// siteIconTarget 回源目标：上游地址优先，没有才落到站点域名。
type siteIconTarget struct {
	// Scheme 显式声明的协议；空表示按探测顺序决定
	Scheme string
	// HostPort 含端口的目标地址，如 192.168.50.100:8006
	HostPort string
}

// key 作为图标缓存键：同一目标换协议命中同一份结果没有意义，协议要进 key。
func (t siteIconTarget) key() string {
	if t.Scheme == "" {
		return "//" + t.HostPort
	}
	return t.Scheme + "://" + t.HostPort
}

// resolveSiteIcon 返回站点图标的 data URL；抓不到返回空串，由前端回落字母头像。
// 抓到后写入 caddy_site_icons，换浏览器打开也能直接复用。
func resolveSiteIcon(ctx context.Context, db *gorm.DB, rawUpstream, rawHost string) string {
	target := parseSiteIconTarget(rawUpstream, rawHost)
	if target == nil {
		return ""
	}
	if ctx == nil {
		ctx = context.Background()
	}

	icon := loadPersistedSiteIcon(db, normalizeSiteIconHost(rawHost))
	if icon != "" {
		return icon
	}

	if icon, ok := readSiteIconCache(target.key()); ok {
		return icon
	}

	icon = fetchSiteIcon(ctx, target)
	writeSiteIconCache(target.key(), icon)
	if db != nil && icon != "" {
		persistSiteIcon(db, normalizeSiteIconHost(rawHost), icon)
	}
	return icon
}

// loadPersistedSiteIcon 读库内已收录的图标；无库或未收录时返回空串。
func loadPersistedSiteIcon(db *gorm.DB, host string) string {
	if db == nil || host == "" {
		return ""
	}
	row, err := caddymodel.NewSiteIconModel(db).FindByHost(context.Background(), host)
	if err != nil || row == nil {
		return ""
	}
	return row.Icon
}

// persistSiteIcon 落库；写失败只记日志，不影响本次返回。
func persistSiteIcon(db *gorm.DB, host, icon string) {
	if db == nil || host == "" {
		return
	}
	if err := caddymodel.NewSiteIconModel(db).Upsert(context.Background(), host, icon); err != nil {
		logx.Errorf("保存站点图标失败 host=%s: %v", host, err)
	}
}

// findUpstreamByHost 从已接管的 Caddy 配置中查找指定 host 对应的反代上游地址。
// 优先解析结构化 modules（含上游池映射）；若未命中再扫描 Caddyfile 原文站点块。
func findUpstreamByHost(db *gorm.DB, rawHost string) string {
	if db == nil {
		return ""
	}
	host := normalizeSiteIconHost(rawHost)
	if host == "" {
		return ""
	}

	config, modulesRaw := caddymodel.LoadConfig(db)

	// 1. 尝试从结构化 modules 查找
	if strings.TrimSpace(modulesRaw) != "" && modulesRaw != caddymodel.EmptyModulesJSON {
		var mod caddyFormModel
		if err := json.Unmarshal([]byte(modulesRaw), &mod); err == nil {
			poolMap := make(map[string]string)
			for _, pool := range mod.Upstreams {
				if len(pool.Targets) > 0 && strings.TrimSpace(pool.Targets[0]) != "" {
					poolMap[pool.Name] = strings.TrimSpace(pool.Targets[0])
				}
			}

			for _, site := range mod.Sites {
				matched := false
				for _, d := range site.Domains {
					if normalizeSiteIconHost(d) == host {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
				for _, r := range site.Routes {
					for _, h := range r.Handles {
						if h.Type == "reverse_proxy" && strings.TrimSpace(h.Upstream) != "" {
							up := strings.TrimSpace(h.Upstream)
							if target, ok := poolMap[up]; ok {
								return target
							}
							return up
						}
					}
				}
			}
		}
	}

	// 2. 尝试从 Caddyfile 原文扫描站点块
	if strings.TrimSpace(config) != "" {
		if up := parseUpstreamFromCaddyfile(config, host); up != "" {
			return up
		}
	}

	return ""
}

var reverseProxyRegex = regexp.MustCompile(`(?m)^\s*reverse_proxy\s+([^\s{]+)`)

// parseUpstreamFromCaddyfile 扫描 Caddyfile 原文，提取包含 targetHost 站点块的首条 reverse_proxy 目标。
func parseUpstreamFromCaddyfile(caddyfile, targetHost string) string {
	lines := strings.Split(caddyfile, "\n")
	inTargetBlock := false
	braceDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if braceDepth == 0 {
			if strings.HasSuffix(trimmed, "{") {
				header := strings.TrimSuffix(trimmed, "{")
				domains := strings.Fields(strings.ReplaceAll(header, ",", " "))
				inTargetBlock = false
				for _, d := range domains {
					if normalizeSiteIconHost(d) == targetHost {
						inTargetBlock = true
						break
					}
				}
				braceDepth = 1
			}
		} else {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if inTargetBlock && braceDepth >= 1 {
				if match := reverseProxyRegex.FindStringSubmatch(line); len(match) > 1 {
					up := strings.TrimSpace(match[1])
					if up != "{" && up != "" {
						return up
					}
				}
			}
			if braceDepth <= 0 {
				braceDepth = 0
				inTargetBlock = false
			}
		}
	}
	return ""
}

// parseSiteIconTarget 选出可回源的目标，取不到返回 nil。
// 负载均衡上游是空格分隔的多个地址，取第一个即可代表站点。
func parseSiteIconTarget(rawUpstream, rawHost string) *siteIconTarget {
	if fields := strings.Fields(rawUpstream); len(fields) > 0 {
		target := fields[0]
		lower := strings.ToLower(target)
		scheme := ""
		if strings.HasPrefix(lower, "https://") {
			scheme, target = "https", target[len("https://"):]
		} else if strings.HasPrefix(lower, "http://") {
			scheme, target = "http", target[len("http://"):]
		}
		// 协议前缀剥掉后斜杠就是路径（localhost:8080/prefix）
		if idx := strings.IndexByte(target, '/'); idx >= 0 {
			target = target[:idx]
		}
		// 要求带端口或带点：unix//、h2c:// 这类非 http 上游拆出来的短名在这里被挡掉
		if strings.ContainsAny(target, ".:") && isValidSiteIconHost(target) {
			return &siteIconTarget{Scheme: scheme, HostPort: target}
		}
	}

	host := normalizeSiteIconHost(rawHost)
	if host == "" {
		return nil
	}
	return &siteIconTarget{HostPort: host}
}

// isValidSiteIconHost 校验主机地址：允许 IPv4/IPv6、域名与 localhost，拒绝带分隔符的输入。
func isValidSiteIconHost(target string) bool {
	if strings.ContainsAny(target, " \t\r\n/\\@\"'") {
		return false
	}
	host := strings.Trim(target, "[]")
	if h, port, err := net.SplitHostPort(target); err == nil {
		// 端口不能为空：h2c://、unix// 这类非 http 上游剥完路径只剩 "h2c:"
		if port == "" {
			return false
		}
		host = strings.Trim(h, "[]")
	}
	if host == "" {
		return false
	}
	if host == "localhost" || net.ParseIP(host) != nil {
		return true
	}
	return siteIconHostPattern.MatchString(host)
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

// fetchSiteIcon 按协议探测图标：显式声明协议的排在前面，未声明时内网多为纯 http。
func fetchSiteIcon(ctx context.Context, target *siteIconTarget) string {
	deadline := time.Now().Add(siteIconTotalBudget)
	schemes := [2]string{"http", "https"}
	if target.Scheme == "https" {
		schemes = [2]string{"https", "http"}
	}
	for _, scheme := range schemes {
		// 目标能连上就算探完了：连得上却没图标，换个协议也一样没有
		if icon, reachable := probeSiteIcon(ctx, scheme+"://"+target.HostPort, deadline); icon != "" || reachable {
			return icon
		}
	}
	return ""
}

// probeSiteIcon 探测一个协议下的图标；reachable 表示目标本身是否可访问。
// 两次请求共享同一个总预算，服务慢时不会把整个请求拖成超时。
func probeSiteIcon(ctx context.Context, baseURL string, deadline time.Time) (string, bool) {
	timeoutCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	if icon := requestSiteIcon(timeoutCtx, baseURL+"/favicon.ico"); icon != "" {
		return icon, true
	}
	// 第二步不能省：SPA 站点的图标多是带 hash 的静态文件名，根路径往往 404
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

func readSiteIconCache(key string) (string, bool) {
	siteIconCache.Lock()
	defer siteIconCache.Unlock()
	item, ok := siteIconCache.items[key]
	if !ok || time.Now().After(item.expires) {
		return "", false
	}
	return item.icon, true
}

func writeSiteIconCache(key, icon string) {
	siteIconCache.Lock()
	defer siteIconCache.Unlock()
	// 失效项按需清掉，避免地图无限增长
	now := time.Now()
	for k, item := range siteIconCache.items {
		if now.After(item.expires) {
			delete(siteIconCache.items, k)
		}
	}
	siteIconCache.items[key] = siteIconCacheItem{icon: icon, expires: now.Add(siteIconCacheTTL)}
}
