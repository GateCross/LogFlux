package notification

import (
	"logflux/common/result"
	"logflux/internal/logic/notification"
	"logflux/internal/svc"
	"net/http"
)

func GetUnreadNotificationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notification.NewGetUnreadNotificationsLogic(r.Context(), svcCtx)
		resp, err := l.GetUnreadNotifications()
		result.HttpResult(r, w, resp, err)
	}
}
