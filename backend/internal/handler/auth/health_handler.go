package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"logflux/internal/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"gorm.io/gorm"
)

// 数据库连通性探测的最长等待时间。
const healthPingTimeout = 3 * time.Second

// 探测数据库连通性：可连通返回 200，探测失败返回 503。
// 状态码由端点直接写出，异常时 curl -f 调用会失败。
func HealthHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pingDatabase(r.Context(), db); err != nil {
			logx.Errorf("健康检查数据库探测失败: %v", err)
			httpx.WriteJson(w, http.StatusServiceUnavailable, response.Result{
				Code:    http.StatusServiceUnavailable,
				Message: "数据库连接异常",
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, response.Result{
			Code:    http.StatusOK,
			Message: "服务正常",
		})
	}
}

func pingDatabase(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("数据库连接未初始化")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, healthPingTimeout)
	defer cancel()

	return sqlDB.PingContext(ctx)
}
