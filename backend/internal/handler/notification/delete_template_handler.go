package notification

import (
	"logflux/common/result"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"logflux/internal/logic/notification"
	"logflux/internal/svc"
	"logflux/internal/types"
)

func DeleteTemplateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IDReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := notification.NewDeleteTemplateLogic(r.Context(), svcCtx)
		resp, err := l.DeleteTemplate(&req)
		result.HttpResult(r, w, resp, err)
	}
}
