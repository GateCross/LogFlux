package dashboard

import (
	"logflux/common/result"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/dashboard"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func GetDashboardSummaryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DashboardSummaryReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := dashboard.NewGetDashboardSummaryLogic(r.Context(), svcCtx)
		resp, err := l.GetDashboardSummary(&req)
		result.HttpResult(r, w, resp, err)
	}
}
