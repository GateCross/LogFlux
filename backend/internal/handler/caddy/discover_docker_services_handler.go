package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

// 扫描 Docker 容器标签，返回仅会话级的站点候选。
// 只读发现：不调用 /load，不写发现库。
func DiscoverDockerServicesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewDiscoverDockerServicesLogic(r.Context(), svcCtx)
		resp, err := l.DiscoverDockerServices()
		result.HttpResult(r, w, resp, err)
	}
}
