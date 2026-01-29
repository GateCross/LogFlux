package notification

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/notification"
	"logflux/internal/svc"
)

func GetRuleListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notification.NewGetRuleListLogic(r.Context(), svcCtx)
		resp, err := l.GetRuleList()
		result.HttpResult(r, w, resp, err)
	}
}
