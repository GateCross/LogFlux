package role

import (
	"net/http"

	"logflux/internal/logic/role"
	"logflux/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetRoleListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := role.NewGetRoleListLogic(r.Context(), svcCtx)
		resp, err := l.GetRoleList()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
