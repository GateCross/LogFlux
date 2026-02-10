package user

import (
	"net/http"

	"logflux/common/result"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/user"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func ToggleUserStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IDReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewToggleUserStatusLogic(r.Context(), svcCtx)
		resp, err := l.ToggleUserStatus(&req)
		result.HttpResult(r, w, resp, err)
	}
}
