package log

import (
	"logflux/common/result"
	"net/http"

	"logflux/internal/logic/log"
	"logflux/internal/svc"
)

func ClearCaddyLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := log.NewClearCaddyLogsLogic(r.Context(), svcCtx)
		resp, err := l.ClearCaddyLogs()
		result.HttpResult(r, w, resp, err)
	}
}
