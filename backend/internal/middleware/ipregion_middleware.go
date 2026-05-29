package middleware

import (
	_ "embed"
	"net"
	"net/http"
	"strings"
	"sync"

	"logflux/internal/response"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
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
}

func NewIPRegionMiddleware(enabled bool, allowCountries []string) *IPRegionMiddleware {
	if !enabled {
		return &IPRegionMiddleware{enabled: false}
	}

	v4Searcher, err := xdb.NewWithBuffer(xdb.IPv4, ipv4XdbData)
	if err != nil {
		logx.Errorf("ip2region IPv4 初始化失败: %v", err)
		return &IPRegionMiddleware{enabled: false}
	}

	v6Searcher, err := xdb.NewWithBuffer(xdb.IPv6, ipv6XdbData)
	if err != nil {
		logx.Errorf("ip2region IPv6 初始化失败: %v", err)
		return &IPRegionMiddleware{enabled: false}
	}

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

	logx.Infof("IP 区域中间件已启用，允许访问的地区: %v", allowCountries)
	return &IPRegionMiddleware{
		v4Searcher: v4Searcher,
		v6Searcher: v6Searcher,
		enabled:    true,
		allowList:  allow,
	}
}

// Reload 热更新 IP 区域配置（并发安全）
func (m *IPRegionMiddleware) Reload(enabled bool, allowCountries []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = enabled
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

func (m *IPRegionMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// forward_auth 端点自身不检查，避免循环拦截
		if r.URL.Path == "/api/internal/geo-check" {
			next(w, r)
			return
		}

		m.mu.RLock()
		enabled := m.enabled
		allowList := m.allowList
		m.mu.RUnlock()

		if !enabled {
			next(w, r)
			return
		}

		ip := getRealIP(r)
		if ip == "" || ip == "127.0.0.1" || ip == "::1" {
			next(w, r)
			return
		}

		region, err := m.search(ip)
		if err != nil {
			// fail-open: 查询失败时放行，避免数据库异常导致全站不可用
			logx.Errorf("ip2region 查询失败: ip=%s err=%v", ip, err)
			next(w, r)
			return
		}

		if region == "" {
			httpx.WriteJson(w, http.StatusForbidden, response.Error(403, "禁止访问"))
			return
		}

		// region 格式: "中国|0|广东省|深圳市|电信"
		country := region
		if idx := strings.IndexByte(region, '|'); idx > 0 {
			country = region[:idx]
		}

		if _, ok := allowList[country]; !ok {
			httpx.WriteJson(w, http.StatusForbidden, response.Error(403, "禁止访问"))
			return
		}

		next(w, r)
	}
}

func (m *IPRegionMiddleware) search(ip string) (string, error) {
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
