package notification

import (
	"net/http"

	"logflux/common/result"
	"logflux/internal/logic/notification"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func TestChannelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TestChannelReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := notification.NewTestChannelLogic(r.Context(), svcCtx)
		resp, err := l.TestChannel(&req)
		result.HttpResult(r, w, resp, err)
	}
}
