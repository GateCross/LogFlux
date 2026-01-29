package notification

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/notification"
	"logflux/internal/svc"
)

func GetTemplateListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notification.NewGetTemplateListLogic(r.Context(), svcCtx)
		resp, err := l.GetTemplateList()
		result.HttpResult(r, w, resp, err)
	}
}
