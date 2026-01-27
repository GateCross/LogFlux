package user

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/user"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetUserListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UserListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewGetUserListLogic(r.Context(), svcCtx)
		resp, err := l.GetUserList(&req)
		result.HttpResult(r, w, resp, err)
	}
}
