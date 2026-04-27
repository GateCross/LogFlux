package log

import (
	"logflux/common/result"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/log"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func ListLogSourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LogSourceListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := log.NewListLogSourcesLogic(r.Context(), svcCtx)
		resp, err := l.ListLogSources(&req)
		result.HttpResult(r, w, resp, err)
	}
}
