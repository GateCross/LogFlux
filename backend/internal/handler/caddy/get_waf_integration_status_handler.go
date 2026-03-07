package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func GetWafIntegrationStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewGetWafIntegrationStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetWafIntegrationStatus()
		result.HttpResult(r, w, resp, err)
	}
}
