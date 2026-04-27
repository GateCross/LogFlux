package notification

import (
	"logflux/common/result"
	"logflux/internal/logic/notification"
	"logflux/internal/svc"
	"net/http"
)

func ClearNotificationLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notification.NewClearNotificationLogsLogic(r.Context(), svcCtx)
		resp, err := l.ClearNotificationLogs()
		result.HttpResult(r, w, resp, err)
	}
}
