package role

import (
	"logflux/common/result"
	"net/http"

	"logflux/internal/logic/role"
	"logflux/internal/svc"
)

func GetRoleListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := role.NewGetRoleListLogic(r.Context(), svcCtx)
		resp, err := l.GetRoleList()
		result.HttpResult(r, w, resp, err)
	}
}
