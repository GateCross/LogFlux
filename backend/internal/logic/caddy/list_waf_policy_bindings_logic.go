package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListWafPolicyBindingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafPolicyBindingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafPolicyBindingsLogic {
	return &ListWafPolicyBindingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafPolicyBindingsLogic) ListWafPolicyBindings(req *types.WafPolicyBindingListReq) (resp *types.WafPolicyBindingListResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()
	if req == nil {
		req = &types.WafPolicyBindingListReq{}
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := l.svcCtx.DB.Model(&model.WafPolicyBinding{})
	if req.PolicyId > 0 {
		db = db.Where("policy_id = ?", req.PolicyId)
	}
	if scopeType := strings.TrimSpace(req.ScopeType); scopeType != "" {
		normalizedScopeType := normalizePolicyScopeType(scopeType)
		if err := validatePolicyScopeType(normalizedScopeType); err != nil {
			return nil, err
		}
		db = db.Where("scope_type = ?", normalizedScopeType)
	}
	if keyword := strings.TrimSpace(req.Name); keyword != "" {
		db = db.Where("name ILIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count policy bindings failed: %w", err)
	}

	var bindings []model.WafPolicyBinding
	offset := (page - 1) * pageSize
	if err := db.Order("priority asc, updated_at desc, id desc").Limit(pageSize).Offset(offset).Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("query policy bindings failed: %w", err)
	}

	items := make([]types.WafPolicyBindingItem, 0, len(bindings))
	for _, binding := range bindings {
		items = append(items, types.WafPolicyBindingItem{
			ID:          binding.ID,
			PolicyId:    binding.PolicyID,
			Name:        binding.Name,
			Description: binding.Description,
			Enabled:     binding.Enabled,
			ScopeType:   binding.ScopeType,
			Host:        binding.Host,
			Path:        binding.Path,
			Method:      binding.Method,
			Priority:    binding.Priority,
			CreatedAt:   formatTime(binding.CreatedAt),
			UpdatedAt:   formatTime(binding.UpdatedAt),
		})
	}

	return &types.WafPolicyBindingListResp{List: items, Total: total}, nil
}
