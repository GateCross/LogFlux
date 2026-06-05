package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"logflux/internal/utils/safego"
	caddymodel "logflux/model/caddy"
	ingestmodel "logflux/model/ingest"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// IPRegionMiddleware 基于 ip2region 的 IP 区域访问控制中间件
type IPRegionMiddleware struct {
	v4Searcher *xdb.Searcher
	v6Searcher *xdb.Searcher
	mu         sync.RWMutex
	enabled    bool
	allowList  map[string]struct{} // 允许的国家、省份或城市集合
	db         *gorm.DB
	logModel   caddymodel.CaddyLogModel
}

func NewIPRegionMiddleware(enabled bool, allowCountries []string, db *gorm.DB) *IPRegionMiddleware {
	var logModel caddymodel.CaddyLogModel
	if db != nil {
		logModel = caddymodel.NewCaddyLogModel(db)
	}

	m := &IPRegionMiddleware{enabled: enabled, db: db, logModel: logModel}
	m.initSearchers()

	m.allowList = buildRegionAllowSet(allowCountries)

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

	ipv4XdbData := loadXdbData("ip2region_v4.xdb")
	ipv6XdbData := loadXdbData("ip2region_v6.xdb")
	logx.Infof("ip2region xdb 数据大小: v4=%d v6=%d", len(ipv4XdbData), len(ipv6XdbData))
	if len(ipv4XdbData) == 0 || len(ipv6XdbData) == 0 {
		logx.Error("ip2region xdb 数据文件缺失")
		return
	}

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

func loadXdbData(filename string) []byte {
	switch filename {
	case "ip2region_v4.xdb":
		if len(embeddedIPv4XdbData) > 0 {
			return embeddedIPv4XdbData
		}
	case "ip2region_v6.xdb":
		if len(embeddedIPv6XdbData) > 0 {
			return embeddedIPv6XdbData
		}
	}

	// 本地开发允许先编译后下载 xdb，运行时从源码目录兜底读取。
	diskPath := filepath.Join("internal", "middleware", "data", filename)
	data, err := os.ReadFile(diskPath)
	if err != nil {
		return nil
	}
	return data
}

// Reload 热更新 IP 区域配置（并发安全）
func (m *IPRegionMiddleware) Reload(enabled bool, allowCountries []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = enabled
	m.initSearchers()
	m.allowList = buildRegionAllowSet(allowCountries)
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
		ip, region, blocked := m.prepareAccess(r)
		if blocked {
			m.logAccess(r, ip, region, http.StatusForbidden, 0)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// 包装 ResponseWriter 捕获状态码和大小
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next(rec, r)

		m.logAccess(r, ip, region, rec.statusCode, rec.written)
	}
}

func (m *IPRegionMiddleware) prepareAccess(r *http.Request) (ip, region string, blocked bool) {
	defer func() {
		if rec := recover(); rec != nil {
			logx.Errorf("IPRegionCheck panic: %v, 放行请求: %s", rec, r.URL.Path)
			blocked = false
		}
	}()

	ip = getRealIP(r)
	region = m.lookupRegion(ip)
	if m.isAllowed(ip, region) {
		return ip, region, false
	}
	return ip, region, true
}

func (m *IPRegionMiddleware) isAllowed(ip, region string) bool {
	m.mu.RLock()
	enabled := m.enabled
	allowList := m.allowList
	m.mu.RUnlock()

	if !enabled || isPrivateIP(ip) {
		return true
	}
	country, province, city := parseRegionParts(region)
	if country == "" {
		return false
	}
	return isRegionAllowed(allowList, country, province, city)
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

func buildRegionAllowSet(regions []string) map[string]struct{} {
	allow := make(map[string]struct{}, len(regions))
	for _, region := range regions {
		region = normalizeRegionRule(region)
		if region != "" {
			allow[region] = struct{}{}
		}
	}
	if len(allow) == 0 {
		allow["中国"] = struct{}{}
	}
	return allow
}

func normalizeRegionRule(region string) string {
	region = strings.TrimSpace(region)
	region = strings.Trim(region, "/")
	if region == "" {
		return ""
	}

	parts := strings.Split(region, "/")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	return strings.Join(normalized, "/")
}

func isRegionAllowed(allowList map[string]struct{}, country, province, city string) bool {
	if _, ok := allowList[country]; ok {
		return true
	}
	if province != "" {
		if _, ok := allowList[country+"/"+province]; ok {
			return true
		}
	}
	if province != "" && city != "" {
		if _, ok := allowList[country+"/"+province+"/"+city]; ok {
			return true
		}
	}
	return false
}

func parseRegionParts(region string) (country, province, city string) {
	// ip2region xdb 实际格式: 国家|省份|城市|运营商|国家代码
	// 例如: 中国|四川省|绵阳市|电信|CN
	parts := strings.Split(region, "|")
	if len(parts) >= 1 {
		country = parts[0]
	}
	if len(parts) >= 3 {
		province = parts[1]
		city = parts[2]
	}
	return
}

func (m *IPRegionMiddleware) logAccess(r *http.Request, ip, region string, status int, size int64) {
	if m.db == nil {
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

	// 内网 IP → system_logs，外网 IP → caddy_logs
	if isPrivateIP(ip) {
		m.logInternalAccess(ip, host, method, uri, r.Proto, status, size, r.UserAgent(), r.RemoteAddr, country, province, city)
		return
	}

	if m.logModel == nil {
		return
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

	safego.New(context.Background(), "写入代理访问日志").Go(func() {
		if err := m.logModel.Create(context.Background(), entry); err != nil {
			logx.Errorf("写入代理访问日志失败: %v", err)
		}
	})
}

func (m *IPRegionMiddleware) logInternalAccess(ip, host, method, uri, proto string, status int, size int64, userAgent, remoteAddr, country, province, city string) {
	extraData := "{}"
	if data, err := json.Marshal(map[string]any{
		"clientIP": ip, "host": host, "method": method, "uri": uri, "proto": proto,
		"status": status, "size": size, "userAgent": userAgent, "remoteIP": remoteAddr,
		"country": country, "province": province, "city": city,
	}); err == nil {
		extraData = string(data)
	}

	level := "info"
	if status >= 500 {
		level = "error"
	} else if status >= 400 {
		level = "warn"
	}

	entry := &ingestmodel.SystemLog{
		LogTime:   time.Now(),
		Level:     level,
		Message:   fmt.Sprintf("Caddy 内网访问 %s %s %d host=%s client=%s", method, uri, status, host, ip),
		Source:    "caddy_internal",
		RawLog:    "{}",
		ExtraData: extraData,
	}

	safego.New(context.Background(), "写入内网访问日志").Go(func() {
		if err := m.db.Create(entry).Error; err != nil {
			logx.Errorf("写入内网访问日志失败: %v", err)
		}
	})
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
		if m.v4Searcher == nil {
			return "", fmt.Errorf("ip2region IPv4 searcher 未初始化")
		}
		return m.v4Searcher.Search(ipBytes)
	}
	if m.v6Searcher == nil {
		return "", fmt.Errorf("ip2region IPv6 searcher 未初始化")
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
