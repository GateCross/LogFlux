package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func GetCaddyServerStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewGetCaddyServerStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetCaddyServerStatus()
		result.HttpResult(r, w, resp, err)
	}
}
