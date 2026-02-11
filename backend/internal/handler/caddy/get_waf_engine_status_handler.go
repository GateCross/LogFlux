package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func GetWafEngineStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewGetWafEngineStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetWafEngineStatus()
		result.HttpResult(r, w, resp, err)
	}
}
