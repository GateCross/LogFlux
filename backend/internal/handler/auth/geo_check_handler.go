package auth

import (
	"net/http"

	"logflux/internal/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// GeoCheckHandler 供 Caddy forward_auth 调用的地理位置校验端点。
// 请求到达此处说明已通过 IPRegionCheck 中间件，直接返回 200。
func GeoCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.OkJsonCtx(r.Context(), w, response.Success(nil))
	}
}
