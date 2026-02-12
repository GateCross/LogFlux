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

type ListWafRuleExclusionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafRuleExclusionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafRuleExclusionsLogic {
	return &ListWafRuleExclusionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafRuleExclusionsLogic) ListWafRuleExclusions(req *types.WafRuleExclusionListReq) (resp *types.WafRuleExclusionListResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()
	if req == nil {
		req = &types.WafRuleExclusionListReq{}
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := l.svcCtx.DB.Model(&model.WafRuleExclusion{})
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
		return nil, fmt.Errorf("count policy exclusions failed: %w", err)
	}

	var exclusions []model.WafRuleExclusion
	offset := (page - 1) * pageSize
	if err := db.Order("updated_at desc, id desc").Limit(pageSize).Offset(offset).Find(&exclusions).Error; err != nil {
		return nil, fmt.Errorf("query policy exclusions failed: %w", err)
	}

	items := make([]types.WafRuleExclusionItem, 0, len(exclusions))
	for _, exclusion := range exclusions {
		items = append(items, types.WafRuleExclusionItem{
			ID:          exclusion.ID,
			PolicyId:    exclusion.PolicyID,
			Name:        exclusion.Name,
			Description: exclusion.Description,
			Enabled:     exclusion.Enabled,
			ScopeType:   exclusion.ScopeType,
			Host:        exclusion.Host,
			Path:        exclusion.Path,
			Method:      exclusion.Method,
			RemoveType:  exclusion.RemoveType,
			RemoveValue: exclusion.RemoveValue,
			CreatedAt:   formatTime(exclusion.CreatedAt),
			UpdatedAt:   formatTime(exclusion.UpdatedAt),
		})
	}

	return &types.WafRuleExclusionListResp{List: items, Total: total}, nil
}
