package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func ClearWafJobsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewClearWafJobsLogic(r.Context(), svcCtx)
		resp, err := l.ClearWafJobs()
		result.HttpResult(r, w, resp, err)
	}
}
