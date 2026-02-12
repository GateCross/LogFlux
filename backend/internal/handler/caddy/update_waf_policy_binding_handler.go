package caddy

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateWafPolicyBindingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WafPolicyBindingUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := caddy.NewUpdateWafPolicyBindingLogic(r.Context(), svcCtx)
		resp, err := l.UpdateWafPolicyBinding(&req)
		result.HttpResult(r, w, resp, err)
	}
}
