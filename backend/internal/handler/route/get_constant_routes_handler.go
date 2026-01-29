package route

import (
	"net/http"

	"logflux/internal/logic/route"
	"logflux/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetConstantRoutesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := route.NewGetConstantRoutesLogic(r.Context(), svcCtx)
		resp, err := l.GetConstantRoutes()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
