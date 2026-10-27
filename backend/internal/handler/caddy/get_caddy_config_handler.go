package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func GetCaddyConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewGetCaddyConfigLogic(r.Context(), svcCtx)
		resp, err := l.GetCaddyConfig()
		result.HttpResult(r, w, resp, err)
	}
}
