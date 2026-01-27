package route

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/route"
	"logflux/internal/svc"
)

func GetUserRoutesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := route.NewGetUserRoutesLogic(r.Context(), svcCtx)
		resp, err := l.GetUserRoutes()
		result.HttpResult(r, w, resp, err)
	}
}
