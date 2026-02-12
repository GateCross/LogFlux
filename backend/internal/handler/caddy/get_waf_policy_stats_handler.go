package caddy

import (
	"net/http"

	"logflux/common/result"
	logiccaddy "logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetWafPolicyStatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WafPolicyStatsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logiccaddy.NewGetWafPolicyStatsLogic(r.Context(), svcCtx)
		resp, err := l.GetWafPolicyStats(&req)
		result.HttpResult(r, w, resp, err)
	}
}
