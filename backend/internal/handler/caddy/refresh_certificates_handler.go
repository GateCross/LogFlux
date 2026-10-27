package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func RefreshCertificatesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewRefreshCertificatesLogic(r.Context(), svcCtx)
		resp, err := l.RefreshCertificates()
		result.HttpResult(r, w, resp, err)
	}
}
