package middleware

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	caddymodel "logflux/model/caddy"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

//go:embed data/ip2region_v4.xdb
var ipv4XdbData []byte

//go:embed data/ip2region_v6.xdb
var ipv6XdbData []byte

// IPRegionMiddleware 基于 ip2region 的 IP 区域访问控制中间件
type IPRegionMiddleware struct {
	v4Searcher *xdb.Searcher
	v6Searcher *xdb.Searcher
	mu         sync.RWMutex
	enabled    bool
	allowList  map[string]struct{} // 允许的国家/地区集合
	logModel   caddymodel.CaddyLogModel
}

func NewIPRegionMiddleware(enabled bool, allowCountries []string, db *gorm.DB) *IPRegionMiddleware {
	var logModel caddymodel.CaddyLogModel
	if db != nil {
		logModel = caddymodel.NewCaddyLogModel(db)
	}

	m := &IPRegionMiddleware{enabled: enabled, logModel: logModel}
	m.initSearchers()

	allow := make(map[string]struct{}, len(allowCountries))
	for _, c := range allowCountries {
		c = strings.TrimSpace(c)
		if c != "" {
			allow[c] = struct{}{}
		}
	}
	if len(allow) == 0 {
		allow["中国"] = struct{}{}
	}
	m.allowList = allow

	if enabled {
		logx.Infof("IP 区域中间件已启用，允许访问的地区: %v", allowCountries)
	} else {
		logx.Info("IP 区域中间件已初始化（拦截未启用，日志记录正常）")
	}
	return m
}

// initSearchers 初始化 ip2region xdb 搜索器（始终初始化，用于日志地理信息查询）
func (m *IPRegionMiddleware) initSearchers() {
	if m.v4Searcher != nil && m.v6Searcher != nil {
		return
	}

	logx.Infof("ip2region xdb 数据大小: v4=%d v6=%d", len(ipv4XdbData), len(ipv6XdbData))

	v4Searcher, err := xdb.NewWithBuffer(xdb.IPv4, ipv4XdbData)
	if err != nil {
		logx.Errorf("ip2region IPv4 初始化失败: %v", err)
		return
	}

	v6Searcher, err := xdb.NewWithBuffer(xdb.IPv6, ipv6XdbData)
	if err != nil {
		logx.Errorf("ip2region IPv6 初始化失败: %v", err)
		return
	}

	m.v4Searcher = v4Searcher
	m.v6Searcher = v6Searcher
}

// Reload 热更新 IP 区域配置（并发安全）
func (m *IPRegionMiddleware) Reload(enabled bool, allowCountries []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = enabled
	m.initSearchers()
	allow := make(map[string]struct{}, len(allowCountries))
	for _, c := range allowCountries {
		c = strings.TrimSpace(c)
		if c != "" {
			allow[c] = struct{}{}
		}
	}
	if len(allow) == 0 {
		allow["中国"] = struct{}{}
	}
	m.allowList = allow
	logx.Infof("IP 区域配置已更新: enabled=%v allowList=%v", enabled, allowCountries)
}

// statusRecorder 包装 ResponseWriter，捕获状态码和响应大小
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (m *IPRegionMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// forward_auth 端点：跳过 IP 区域拦截，但记录日志
		isGeoCheck := r.URL.Path == "/api/internal/geo-check"

		if !isGeoCheck {
			defer func() {
				if rec := recover(); rec != nil {
					logx.Errorf("IPRegionCheck panic: %v, 放行请求: %s", rec, r.URL.Path)
					next(w, r)
				}
			}()
		}

		ip := getRealIP(r)
		region := m.lookupRegion(ip)

		// IP 区域访问控制（geo-check 跳过拦截，仅记录日志）
		if !isGeoCheck {
			m.mu.RLock()
			enabled := m.enabled
			allowList := m.allowList
			m.mu.RUnlock()

			if enabled && !isPrivateIP(ip) {
				country := parseCountry(region)
				if country == "" {
					m.logAccess(r, ip, region, http.StatusForbidden, 0)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				if _, ok := allowList[country]; !ok {
					m.logAccess(r, ip, region, http.StatusForbidden, 0)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}
		}

		// 包装 ResponseWriter 捕获状态码和大小
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next(rec, r)

		m.logAccess(r, ip, region, rec.statusCode, rec.written)
	}
}

// lookupRegion 查询 IP 地理信息，失败返回空串
func (m *IPRegionMiddleware) lookupRegion(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return ""
	}
	region, err := m.search(ip)
	if err != nil {
		logx.Errorf("ip2region 查询失败: ip=%s err=%v", ip, err)
		return ""
	}
	return region
}

func parseCountry(region string) string {
	if region == "" {
		return ""
	}
	if idx := strings.IndexByte(region, '|'); idx > 0 {
		return region[:idx]
	}
	return region
}

func parseRegionParts(region string) (country, province, city string) {
	parts := strings.Split(region, "|")
	if len(parts) >= 1 {
		country = parts[0]
	}
	if len(parts) >= 3 {
		province = parts[2]
	}
	if len(parts) >= 4 {
		city = parts[3]
	}
	return
}

func (m *IPRegionMiddleware) logAccess(r *http.Request, ip, region string, status int, size int64) {
	if m.logModel == nil {
		return
	}

	country, province, city := parseRegionParts(region)

	// 优先使用 Caddy forward_auth 传递的原始请求信息
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	uri := r.Header.Get("X-Forwarded-Uri")
	if uri == "" {
		uri = r.URL.RequestURI()
	}
	method := r.Header.Get("X-Forwarded-Method")
	if method == "" {
		method = r.Method
	}

	entry := &caddymodel.CaddyLog{
		LogTime:   time.Now(),
		Country:   country,
		Province:  province,
		City:      city,
		Host:      host,
		Method:    method,
		Uri:       uri,
		Proto:     r.Proto,
		Status:    status,
		Size:      size,
		UserAgent: r.UserAgent(),
		RemoteIP:  r.RemoteAddr,
		ClientIP:  ip,
		RawLog:    "{}",
		ExtraData: "{}",
	}

	go func() {
		if err := m.logModel.Create(context.Background(), entry); err != nil {
			logx.Errorf("写入访问日志失败: %v", err)
		}
	}()
}

func (m *IPRegionMiddleware) search(ip string) (region string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ip2region search panic: %v", r)
		}
	}()

	ipBytes, err := xdb.ParseIP(ip)
	if err != nil {
		return "", err
	}

	if len(ipBytes) == 4 {
		return m.v4Searcher.Search(ipBytes)
	}
	return m.v6Searcher.Search(ipBytes)
}

// getRealIP 获取客户端真实 IP，支持 IPv4 和 IPv6
func getRealIP(r *http.Request) string {
	// Cloudflare 环境优先使用 CF-Connecting-IP
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		if idx := strings.IndexByte(ip, ','); idx > 0 {
			return strings.TrimSpace(ip[:idx])
		}
		return strings.TrimSpace(ip)
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isPrivateIP 判断 IP 是否为内网/本地地址，内网 IP 不做区域拦截
func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() {
		return true
	}
	return false
}
