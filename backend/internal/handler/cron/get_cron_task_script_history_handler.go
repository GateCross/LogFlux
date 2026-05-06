package cron

import (
	"logflux/common/result"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/cron"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func GetCronTaskScriptHistoryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CronTaskFileListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cron.NewGetCronTaskScriptHistoryLogic(r.Context(), svcCtx)
		resp, err := l.GetCronTaskScriptHistory(&req)
		result.HttpResult(r, w, resp, err)
	}
}
