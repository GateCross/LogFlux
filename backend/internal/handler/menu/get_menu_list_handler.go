package menu

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/menu"
	"logflux/internal/svc"
)

func GetMenuListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := menu.NewGetMenuListLogic(r.Context(), svcCtx)
		resp, err := l.GetMenuList()
		result.HttpResult(r, w, resp, err)
	}
}
