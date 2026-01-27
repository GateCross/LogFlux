package route

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/route"
	"logflux/internal/svc"
)

func GetConstantRoutesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := route.NewGetConstantRoutesLogic(r.Context(), svcCtx)
		resp, err := l.GetConstantRoutes()
		result.HttpResult(r, w, resp, err)
	}
}
