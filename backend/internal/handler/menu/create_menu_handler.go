package menu

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/menu"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateMenuHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateMenuReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := menu.NewCreateMenuLogic(r.Context(), svcCtx)
		resp, err := l.CreateMenu(&req)
		result.HttpResult(r, w, resp, err)
	}
}
