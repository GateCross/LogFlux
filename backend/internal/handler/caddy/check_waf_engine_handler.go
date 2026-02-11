package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func CheckWafEngineHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewCheckWafEngineLogic(r.Context(), svcCtx)
		resp, err := l.CheckWafEngine()
		result.HttpResult(r, w, resp, err)
	}
}
