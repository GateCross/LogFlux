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

type CreateWafRuleExclusionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateWafRuleExclusionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateWafRuleExclusionLogic {
	return &CreateWafRuleExclusionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateWafRuleExclusionLogic) CreateWafRuleExclusion(req *types.WafRuleExclusionReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil {
		return nil, fmt.Errorf("invalid policy exclusion payload")
	}
	if err := validatePolicyIDExists(l.svcCtx.DB, req.PolicyId); err != nil {
		return nil, err
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

	exclusion := &model.WafRuleExclusion{
		PolicyID:    req.PolicyId,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Enabled:     true,
		ScopeType:   scopeType,
		Host:        host,
		Path:        path,
		Method:      method,
		RemoveType:  removeType,
		RemoveValue: removeValue,
	}
	if req.Enabled {
		exclusion.Enabled = true
	}

	if err := l.svcCtx.DB.Create(exclusion).Error; err != nil {
		return nil, fmt.Errorf("create policy exclusion failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
