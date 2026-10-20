package middleware

import (
	"encoding/json"
	"errors"
	"net/http"

	"logflux/internal/response"
	usermodel "logflux/model/user"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"gorm.io/gorm"
)

// ActiveUserMiddleware 只校验登录用户存在且未禁用，不做角色 / 权限判断。
// 历史 RBAC 中间件已删除；本中间件保留「禁用用户立即失效」的能力。
type ActiveUserMiddleware struct {
	db *gorm.DB
}

func NewActiveUserMiddleware(db *gorm.DB) *ActiveUserMiddleware {
	return &ActiveUserMiddleware{db: db}
}

func (m *ActiveUserMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.db == nil {
			writeAuthError(w, http.StatusInternalServerError, 500, "认证服务未初始化")
			return
		}

		userID, err := userIDFromContext(r)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, 401, "登录状态无效")
			return
		}

		var user usermodel.User
		if err := m.db.Select("id", "status").First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeAuthError(w, http.StatusUnauthorized, 401, "用户不存在")
				return
			}
			logx.Errorf("查询登录用户失败: userID=%d err=%v", userID, err)
			writeAuthError(w, http.StatusInternalServerError, 500, "认证校验失败")
			return
		}

		if user.Status == 0 {
			writeAuthError(w, http.StatusForbidden, 403, "用户已被禁用")
			return
		}

		next(w, r)
	}
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
		var number json.Number = json.Number(v)
		parsed, err := number.Int64()
		if err != nil || parsed <= 0 {
			return 0, errors.New("用户 ID 无效")
		}
		return uint(parsed), nil
	default:
		return 0, errors.New("缺少用户 ID")
	}
}

func writeAuthError(w http.ResponseWriter, statusCode, code int, msg string) {
	httpx.WriteJson(w, statusCode, response.Error(code, msg))
}
