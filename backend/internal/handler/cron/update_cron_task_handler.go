package cron

import (
	"logflux/common/result"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/cron"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func UpdateCronTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CronTaskUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cron.NewUpdateCronTaskLogic(r.Context(), svcCtx)
		resp, err := l.UpdateCronTask(&req)
		result.HttpResult(r, w, resp, err)
	}
}
