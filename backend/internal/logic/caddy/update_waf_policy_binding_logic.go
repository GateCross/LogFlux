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

type UpdateWafPolicyBindingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateWafPolicyBindingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWafPolicyBindingLogic {
	return &UpdateWafPolicyBindingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateWafPolicyBindingLogic) UpdateWafPolicyBinding(req *types.WafPolicyBindingUpdateReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy binding id is required")
	}
	if err := validatePolicyIDExists(l.svcCtx.DB, req.PolicyId); err != nil {
		return nil, err
	}

	var binding model.WafPolicyBinding
	if err := l.svcCtx.DB.First(&binding, req.ID).Error; err != nil {
		return nil, fmt.Errorf("policy binding not found")
	}

	scopeType, host, path, method, err := normalizeAndValidateBindingScopeFields(req.ScopeType, req.Host, req.Path, req.Method)
	if err != nil {
		return nil, err
	}

	priority := normalizePolicyBindingPriority(req.Priority)
	if err := validatePolicyBindingPriority(priority); err != nil {
		return nil, err
	}

	binding.PolicyID = req.PolicyId
	binding.Name = strings.TrimSpace(req.Name)
	binding.Description = strings.TrimSpace(req.Description)
	binding.Enabled = req.Enabled
	binding.ScopeType = scopeType
	binding.Host = host
	binding.Path = path
	binding.Method = method
	binding.Priority = priority

	if err := validatePolicyBindingConflict(l.svcCtx.DB, &binding); err != nil {
		return nil, err
	}
	if err := l.svcCtx.DB.Save(&binding).Error; err != nil {
		return nil, fmt.Errorf("update policy binding failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
