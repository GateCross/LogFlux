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

type UpdateWafRuleExclusionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateWafRuleExclusionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWafRuleExclusionLogic {
	return &UpdateWafRuleExclusionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateWafRuleExclusionLogic) UpdateWafRuleExclusion(req *types.WafRuleExclusionUpdateReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy exclusion id is required")
	}
	if err := validatePolicyIDExists(l.svcCtx.DB, req.PolicyId); err != nil {
		return nil, err
	}

	var exclusion model.WafRuleExclusion
	if err := l.svcCtx.DB.First(&exclusion, req.ID).Error; err != nil {
		return nil, fmt.Errorf("policy exclusion not found")
	}

	scopeType, host, path, method, err := normalizeAndValidateExclusionScopeFields(req.ScopeType, req.Host, req.Path, req.Method)
	if err != nil {
		return nil, err
	}
	removeType := normalizePolicyRemoveType(req.RemoveType)
	if err := validatePolicyRemoveType(removeType); err != nil {
		return nil, err
	}
	removeValue := strings.TrimSpace(req.RemoveValue)
	if removeValue == "" {
		return nil, fmt.Errorf("remove value is required")
	}

	exclusion.PolicyID = req.PolicyId
	exclusion.Name = strings.TrimSpace(req.Name)
	exclusion.Description = strings.TrimSpace(req.Description)
	exclusion.Enabled = req.Enabled
	exclusion.ScopeType = scopeType
	exclusion.Host = host
	exclusion.Path = path
	exclusion.Method = method
	exclusion.RemoveType = removeType
	exclusion.RemoveValue = removeValue

	if err := l.svcCtx.DB.Save(&exclusion).Error; err != nil {
		return nil, fmt.Errorf("update policy exclusion failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
