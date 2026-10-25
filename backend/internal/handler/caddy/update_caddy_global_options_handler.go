// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func UpdateCaddyGlobalOptionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CaddyGlobalOptionsUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := caddy.NewUpdateCaddyGlobalOptionsLogic(r.Context(), svcCtx)
		resp, err := l.UpdateCaddyGlobalOptions(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
