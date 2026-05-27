package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"logflux/internal/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// RateLimitMiddleware 基于 IP 的速率限制中间件
type RateLimitMiddleware struct {
	attempts map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
	paths    []string // 需要限制的路径前缀
}

// NewRateLimitMiddleware 创建速率限制中间件
// limit: 窗口期内允许的最大请求数
// window: 时间窗口
// paths: 需要限制的路径前缀列表
func NewRateLimitMiddleware(limit int, window time.Duration, paths ...string) *RateLimitMiddleware {
	m := &RateLimitMiddleware{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		paths:    paths,
	}

	// 定期清理过期记录
	go m.cleanup()

	return m
}

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 检查是否需要限制该路径
		if !m.shouldLimit(r.URL.Path) {
			next(w, r)
			return
		}

		ip := getClientIP(r)

		m.mu.Lock()
		now := time.Now()

		// 清理该 IP 的过期记录
		validAttempts := []time.Time{}
		for _, t := range m.attempts[ip] {
			if now.Sub(t) < m.window {
				validAttempts = append(validAttempts, t)
			}
		}

		if len(validAttempts) >= m.limit {
			m.mu.Unlock()
			httpx.WriteJson(w, http.StatusTooManyRequests, response.Error(429, "请求过于频繁，请稍后再试"))
			return
		}

		m.attempts[ip] = append(validAttempts, now)
		m.mu.Unlock()

		next(w, r)
	}
}

// shouldLimit 检查路径是否需要速率限制
func (m *RateLimitMiddleware) shouldLimit(path string) bool {
	if len(m.paths) == 0 {
		return true // 如果没有配置路径，则限制所有请求
	}

	for _, p := range m.paths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// cleanup 定期清理过期的 IP 记录
func (m *RateLimitMiddleware) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for ip, attempts := range m.attempts {
			validAttempts := []time.Time{}
			for _, t := range attempts {
				if now.Sub(t) < m.window {
					validAttempts = append(validAttempts, t)
				}
			}
			if len(validAttempts) == 0 {
				delete(m.attempts, ip)
			} else {
				m.attempts[ip] = validAttempts
			}
		}
		m.mu.Unlock()
	}
}

// getClientIP 获取客户端真实 IP
func getClientIP(r *http.Request) string {
	// 优先从 X-Forwarded-For 获取
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For 可能包含多个 IP，取第一个
		for i, c := range ip {
			if c == ',' {
				return ip[:i]
			}
		}
		return ip
	}

	// 其次从 X-Real-IP 获取
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// 最后使用 RemoteAddr
	return r.RemoteAddr
}
