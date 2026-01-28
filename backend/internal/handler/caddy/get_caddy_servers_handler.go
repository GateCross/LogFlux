package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func GetCaddyServersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewGetCaddyServersLogic(r.Context(), svcCtx)
		resp, err := l.GetCaddyServers()
		result.HttpResult(r, w, resp, err)
	}
}
