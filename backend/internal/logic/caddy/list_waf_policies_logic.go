package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListWafPoliciesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafPoliciesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafPoliciesLogic {
	return &ListWafPoliciesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafPoliciesLogic) ListWafPolicies(req *types.WafPolicyListReq) (resp *types.WafPolicyListResp, err error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := l.svcCtx.DB.Model(&model.WafPolicy{})
	if keyword := strings.TrimSpace(req.Name); keyword != "" {
		db = db.Where("name ILIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count policies failed: %w", err)
	}

	var policies []model.WafPolicy
	offset := (page - 1) * pageSize
	if err := db.Order("updated_at desc, id desc").Limit(pageSize).Offset(offset).Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("query policies failed: %w", err)
	}

	items := make([]types.WafPolicyItem, 0, len(policies))
	for _, policy := range policies {
		configText := ""
		if len(policy.Config) > 0 {
			if configJSON, marshalErr := json.Marshal(policy.Config); marshalErr == nil {
				configText = string(configJSON)
			}
		}

		items = append(items, types.WafPolicyItem{
			ID:                      policy.ID,
			Name:                    policy.Name,
			Description:             policy.Description,
			Enabled:                 policy.Enabled,
			IsDefault:               policy.IsDefault,
			EngineMode:              policy.EngineMode,
			AuditEngine:             policy.AuditEngine,
			AuditLogFormat:          policy.AuditLogFormat,
			AuditRelevantStatus:     policy.AuditRelevantStatus,
			RequestBodyAccess:       policy.RequestBodyAccess,
			RequestBodyLimit:        policy.RequestBodyLimit,
			RequestBodyNoFilesLimit: policy.RequestBodyNoFilesLimit,
			Config:                  configText,
			CreatedAt:               formatTime(policy.CreatedAt),
			UpdatedAt:               formatTime(policy.UpdatedAt),
		})
	}

	return &types.WafPolicyListResp{List: items, Total: total}, nil
}

