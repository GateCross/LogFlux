package route

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/route"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func IsRouteExistHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IsRouteExistReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := route.NewIsRouteExistLogic(r.Context(), svcCtx)
		resp, err := l.IsRouteExist(&req)
		result.HttpResult(r, w, resp, err)
	}
}
