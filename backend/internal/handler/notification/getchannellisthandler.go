package notification

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/notification"
	"logflux/internal/svc"
)

func GetChannelListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notification.NewGetChannelListLogic(r.Context(), svcCtx)
		resp, err := l.GetChannelList()
		result.HttpResult(r, w, resp, err)
	}
}
