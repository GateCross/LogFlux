// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/caddy"
	"logflux/internal/svc"
)

func ResetCaddyEntryBlockHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := caddy.NewResetCaddyEntryBlockLogic(r.Context(), svcCtx)
		resp, err := l.ResetCaddyEntryBlock()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
