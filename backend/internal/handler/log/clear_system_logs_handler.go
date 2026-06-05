package log

import (
	"logflux/common/result"
	"net/http"

	"logflux/internal/logic/log"
	"logflux/internal/svc"
)

func ClearSystemLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := log.NewClearSystemLogsLogic(r.Context(), svcCtx)
		resp, err := l.ClearSystemLogs()
		result.HttpResult(r, w, resp, err)
	}
}
