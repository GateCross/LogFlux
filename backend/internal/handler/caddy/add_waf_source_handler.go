package caddy

import (
	"context"
	"net/http"

	"logflux/common/result"
	logiccaddy "logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func AddWafSourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Name         string `json:"name"`
			Kind         string `json:"kind,optional"`
			Mode         string `json:"mode,optional"`
			Url          string `json:"url,optional"`
			ChecksumUrl  string `json:"checksumUrl,optional"`
			ProxyUrl     string `json:"proxyUrl,optional"`
			AuthType     string `json:"authType,optional"`
			AuthSecret   string `json:"authSecret,optional"`
			Schedule     string `json:"schedule,optional"`
			Enabled      *bool  `json:"enabled,optional"`
			AutoCheck    *bool  `json:"autoCheck,optional"`
			AutoDownload *bool  `json:"autoDownload,optional"`
			AutoActivate *bool  `json:"autoActivate,optional"`
			Meta         string `json:"meta,optional"`
		}
		if err := httpx.Parse(r, &payload); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		req := types.WafSourceReq{
			Name:        payload.Name,
			Kind:        payload.Kind,
			Mode:        payload.Mode,
			Url:         payload.Url,
			ChecksumUrl: payload.ChecksumUrl,
			ProxyUrl:    payload.ProxyUrl,
			AuthType:    payload.AuthType,
			AuthSecret:  payload.AuthSecret,
			Schedule:    payload.Schedule,
			Meta:        payload.Meta,
		}

		boolMask := map[string]bool{
			"enabled":      payload.Enabled != nil,
			"autoCheck":    payload.AutoCheck != nil,
			"autoDownload": payload.AutoDownload != nil,
			"autoActivate": payload.AutoActivate != nil,
		}
		if payload.Enabled != nil {
			req.Enabled = *payload.Enabled
		}
		if payload.AutoCheck != nil {
			req.AutoCheck = *payload.AutoCheck
		}
		if payload.AutoDownload != nil {
			req.AutoDownload = *payload.AutoDownload
		}
		if payload.AutoActivate != nil {
			req.AutoActivate = *payload.AutoActivate
		}

		ctx := context.WithValue(r.Context(), "waf_source_bool_mask", boolMask)
		l := logiccaddy.NewAddWafSourceLogic(ctx, svcCtx)
		resp, err := l.AddWafSource(&req)
		result.HttpResult(r, w, resp, err)
	}
}
