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

func UpdateWafPolicyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			ID                      uint   `path:"id"`
			Name                    string `json:"name,optional"`
			Description             string `json:"description,optional"`
			Enabled                 *bool  `json:"enabled,optional"`
			IsDefault               *bool  `json:"isDefault,optional"`
			EngineMode              string `json:"engineMode,optional"`
			AuditEngine             string `json:"auditEngine,optional"`
			AuditLogFormat          string `json:"auditLogFormat,optional"`
			AuditRelevantStatus     string `json:"auditRelevantStatus,optional"`
			RequestBodyAccess       *bool  `json:"requestBodyAccess,optional"`
			RequestBodyLimit        int64  `json:"requestBodyLimit,optional"`
			RequestBodyNoFilesLimit int64  `json:"requestBodyNoFilesLimit,optional"`
			Config                  string `json:"config,optional"`
		}
		if err := httpx.Parse(r, &payload); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		req := types.WafPolicyUpdateReq{
			ID:                      payload.ID,
			Name:                    payload.Name,
			Description:             payload.Description,
			EngineMode:              payload.EngineMode,
			AuditEngine:             payload.AuditEngine,
			AuditLogFormat:          payload.AuditLogFormat,
			AuditRelevantStatus:     payload.AuditRelevantStatus,
			RequestBodyLimit:        payload.RequestBodyLimit,
			RequestBodyNoFilesLimit: payload.RequestBodyNoFilesLimit,
			Config:                  payload.Config,
		}

		boolMask := map[string]bool{
			"enabled":           payload.Enabled != nil,
			"isDefault":         payload.IsDefault != nil,
			"requestBodyAccess": payload.RequestBodyAccess != nil,
		}
		if payload.Enabled != nil {
			req.Enabled = *payload.Enabled
		}
		if payload.IsDefault != nil {
			req.IsDefault = *payload.IsDefault
		}
		if payload.RequestBodyAccess != nil {
			req.RequestBodyAccess = *payload.RequestBodyAccess
		}

		ctx := context.WithValue(r.Context(), "waf_policy_bool_mask", boolMask)
		l := logiccaddy.NewUpdateWafPolicyLogic(ctx, svcCtx)
		resp, err := l.UpdateWafPolicy(&req)
		result.HttpResult(r, w, resp, err)
	}
}
