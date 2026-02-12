package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateWafRuleExclusionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WafRuleExclusionUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := caddy.NewUpdateWafRuleExclusionLogic(r.Context(), svcCtx)
		resp, err := l.UpdateWafRuleExclusion(&req)
		result.HttpResult(r, w, resp, err)
	}
}
