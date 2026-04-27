package log

import (
	"logflux/common/result"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/log"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func UpdateLogSourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LogSourceUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := log.NewUpdateLogSourceLogic(r.Context(), svcCtx)
		resp, err := l.UpdateLogSource(&req)
		result.HttpResult(r, w, resp, err)
	}
}
