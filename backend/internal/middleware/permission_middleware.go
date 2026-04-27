package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"logflux/internal/response"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"gorm.io/gorm"
)

type permissionRule struct {
	authOnly    bool
	roles       []string
	permissions []string
}

// PermissionMiddleware 基于当前用户角色和权限执行服务端 RBAC 校验。
type PermissionMiddleware struct {
	db *gorm.DB
}

func NewPermissionMiddleware(db *gorm.DB) *PermissionMiddleware {
	return &PermissionMiddleware{db: db}
}

func (m *PermissionMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rule := permissionRuleFor(r.Method, r.URL.Path)
		if rule.authOnly {
			next(w, r)
			return
		}

		if m == nil || m.db == nil {
			writePermissionError(w, http.StatusInternalServerError, 500, "权限服务未初始化")
			return
		}

		userID, err := userIDFromContext(r)
		if err != nil {
			writePermissionError(w, http.StatusUnauthorized, 401, "登录状态无效")
			return
		}

		var user model.User
		if err := m.db.Select("id", "roles", "status").First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writePermissionError(w, http.StatusUnauthorized, 401, "用户不存在")
				return
			}
			logx.Errorf("查询权限用户失败: userID=%d err=%v", userID, err)
			writePermissionError(w, http.StatusInternalServerError, 500, "权限校验失败")
			return
		}

		if user.Status == 0 {
			writePermissionError(w, http.StatusForbidden, 403, "用户已被禁用")
			return
		}

		roleNames := []string(user.Roles)
		if hasAny(roleNames, "admin") || hasAny(roleNames, rule.roles...) {
			next(w, r)
			return
		}

		if len(rule.permissions) > 0 {
			permissions, err := m.loadPermissions(roleNames)
			if err != nil {
				logx.Errorf("查询角色权限失败: userID=%d roles=%v err=%v", userID, roleNames, err)
				writePermissionError(w, http.StatusInternalServerError, 500, "权限校验失败")
				return
			}
			if hasAny(permissions, rule.permissions...) {
				next(w, r)
				return
			}
		}

		writePermissionError(w, http.StatusForbidden, 403, "权限不足")
	}
}

func (m *PermissionMiddleware) loadPermissions(roleNames []string) ([]string, error) {
	if len(roleNames) == 0 {
		return nil, nil
	}

	var roles []model.Role
	if err := m.db.Select("name", "permissions").Where("name IN ?", roleNames).Find(&roles).Error; err != nil {
		return nil, err
	}

	permissions := make([]string, 0)
	seen := make(map[string]struct{})
	for _, role := range roles {
		for _, permission := range role.Permissions {
			permission = strings.TrimSpace(permission)
			if permission == "" {
				continue
			}
			if _, exists := seen[permission]; exists {
				continue
			}
			seen[permission] = struct{}{}
			permissions = append(permissions, permission)
		}
	}
	return permissions, nil
}

func permissionRuleFor(method, path string) permissionRule {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)

	if isAuthOnlyRoute(method, path) {
		return permissionRule{authOnly: true}
	}

	switch {
	case path == "/api/dashboard/summary":
		return permissionRule{permissions: []string{"dashboard"}}
	case path == "/api/caddy/logs":
		return permissionRule{permissions: []string{"logs_caddy", "logs"}}
	case path == "/api/system/logs":
		return permissionRule{permissions: []string{"logs"}}
	case path == "/api/source" && method == http.MethodGet:
		return permissionRule{permissions: []string{"logs"}}
	case strings.HasPrefix(path, "/api/caddy/server") && method == http.MethodGet:
		return permissionRule{roles: []string{"admin", "analyst"}}
	case strings.HasPrefix(path, "/api/caddy/"):
		return permissionRule{roles: []string{"admin"}}
	case strings.HasPrefix(path, "/api/user"):
		return permissionRule{roles: []string{"admin"}}
	case strings.HasPrefix(path, "/api/role"):
		return permissionRule{roles: []string{"admin"}}
	case strings.HasPrefix(path, "/api/menu"):
		return permissionRule{roles: []string{"admin"}}
	case strings.HasPrefix(path, "/api/notification"):
		return permissionRule{roles: []string{"admin"}}
	case strings.HasPrefix(path, "/api/cron"):
		return permissionRule{roles: []string{"admin"}}
	case strings.HasPrefix(path, "/api/source"):
		return permissionRule{roles: []string{"admin"}}
	default:
		return permissionRule{roles: []string{"admin"}}
	}
}

func isAuthOnlyRoute(method, path string) bool {
	if method == http.MethodGet && (path == "/api/user/info" || path == "/api/route/getUserRoutes" || path == "/api/notification/unread") {
		return true
	}
	if method == http.MethodPost && (path == "/api/user/change_password" || path == "/api/notification/read/all") {
		return true
	}
	if method == http.MethodPut && path == "/api/user/preferences" {
		return true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/notification/read/") {
		return true
	}
	return false
}

func userIDFromContext(r *http.Request) (uint, error) {
	if r == nil {
		return 0, errors.New("请求为空")
	}

	value := r.Context().Value("userId")
	switch v := value.(type) {
	case json.Number:
		parsed, err := v.Int64()
		if err != nil || parsed <= 0 {
			return 0, errors.New("用户 ID 无效")
		}
		return uint(parsed), nil
	case float64:
		if v <= 0 {
			return 0, errors.New("用户 ID 无效")
		}
		return uint(v), nil
	case int:
		if v <= 0 {
			return 0, errors.New("用户 ID 无效")
		}
		return uint(v), nil
	case int64:
		if v <= 0 {
			return 0, errors.New("用户 ID 无效")
		}
		return uint(v), nil
	case uint:
		if v == 0 {
			return 0, errors.New("用户 ID 无效")
		}
		return v, nil
	case string:
		var number json.Number = json.Number(strings.TrimSpace(v))
		parsed, err := number.Int64()
		if err != nil || parsed <= 0 {
			return 0, errors.New("用户 ID 无效")
		}
		return uint(parsed), nil
	default:
		return 0, errors.New("缺少用户 ID")
	}
}

func hasAny(values []string, targets ...string) bool {
	if len(values) == 0 || len(targets) == 0 {
		return false
	}

	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	for _, target := range targets {
		if _, exists := set[strings.TrimSpace(target)]; exists {
			return true
		}
	}
	return false
}

func writePermissionError(w http.ResponseWriter, statusCode, code int, msg string) {
	httpx.WriteJson(w, statusCode, response.Error(code, msg))
}
