package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func GetIpRegionConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewGetIpRegionConfigLogic(r.Context(), svcCtx)
		resp, err := l.GetIpRegionConfig()
		result.HttpResult(r, w, resp, err)
	}
}
