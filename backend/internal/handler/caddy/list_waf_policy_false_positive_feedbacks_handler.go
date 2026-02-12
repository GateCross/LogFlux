package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListWafPolicyFalsePositiveFeedbacksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WafPolicyFalsePositiveFeedbackListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := caddy.NewListWafPolicyFalsePositiveFeedbacksLogic(r.Context(), svcCtx)
		resp, err := l.ListWafPolicyFalsePositiveFeedbacks(&req)
		result.HttpResult(r, w, resp, err)
	}
}
