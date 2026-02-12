package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateWafPolicyFalsePositiveFeedbackStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WafPolicyFalsePositiveFeedbackStatusUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := caddy.NewUpdateWafPolicyFalsePositiveFeedbackStatusLogic(r.Context(), svcCtx)
		resp, err := l.UpdateWafPolicyFalsePositiveFeedbackStatus(&req)
		result.HttpResult(r, w, resp, err)
	}
}
