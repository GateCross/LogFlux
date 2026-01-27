package role

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/role"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateRolePermissionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateRolePermissionsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := role.NewUpdateRolePermissionsLogic(r.Context(), svcCtx)
		resp, err := l.UpdateRolePermissions(&req)
		result.HttpResult(r, w, resp, err)
	}
}
