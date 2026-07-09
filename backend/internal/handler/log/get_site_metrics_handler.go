package log

import (
	"logflux/common/result"
	"net/http"

	"logflux/internal/logic/log"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetSiteMetricsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SiteMetricsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := log.NewGetSiteMetricsLogic(r.Context(), svcCtx)
		resp, err := l.GetSiteMetrics(&req)
		result.HttpResult(r, w, resp, err)
	}
}
