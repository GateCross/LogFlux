package menu

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/menu"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateMenuHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateMenuReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := menu.NewUpdateMenuLogic(r.Context(), svcCtx)
		resp, err := l.UpdateMenu(&req)
		result.HttpResult(r, w, resp, err)
	}
}
