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

type CreateWafPolicyBindingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateWafPolicyBindingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateWafPolicyBindingLogic {
	return &CreateWafPolicyBindingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateWafPolicyBindingLogic) CreateWafPolicyBinding(req *types.WafPolicyBindingReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil {
		return nil, fmt.Errorf("invalid policy binding payload")
	}
	if err := validatePolicyIDExists(l.svcCtx.DB, req.PolicyId); err != nil {
		return nil, err
	}

	scopeType, host, path, method, err := normalizeAndValidateBindingScopeFields(req.ScopeType, req.Host, req.Path, req.Method)
	if err != nil {
		return nil, err
	}

	priority := normalizePolicyBindingPriority(req.Priority)
	if err := validatePolicyBindingPriority(priority); err != nil {
		return nil, err
	}

	binding := &model.WafPolicyBinding{
		PolicyID:    req.PolicyId,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Enabled:     true,
		ScopeType:   scopeType,
		Host:        host,
		Path:        path,
		Method:      method,
		Priority:    priority,
	}
	if req.Enabled {
		binding.Enabled = true
	}

	if err := validatePolicyBindingConflict(l.svcCtx.DB, binding); err != nil {
		return nil, err
	}
	if err := l.svcCtx.DB.Create(binding).Error; err != nil {
		return nil, fmt.Errorf("create policy binding failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
