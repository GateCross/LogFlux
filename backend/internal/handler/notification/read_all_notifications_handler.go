package notification

import (
	"logflux/common/result"
	"logflux/internal/logic/notification"
	"logflux/internal/svc"
	"net/http"
)

func ReadAllNotificationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notification.NewReadAllNotificationsLogic(r.Context(), svcCtx)
		resp, err := l.ReadAllNotifications()
		result.HttpResult(r, w, resp, err)
	}
}
