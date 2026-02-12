package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidateWafPolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewValidateWafPolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateWafPolicyLogic {
	return &ValidateWafPolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ValidateWafPolicyLogic) ValidateWafPolicy(req *types.WafPolicyActionReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}

	var policy model.WafPolicy
	if err := l.svcCtx.DB.First(&policy, req.ID).Error; err != nil {
		return nil, fmt.Errorf("policy not found")
	}

	directives, err := buildPolicyDirectivesWithExclusions(l.svcCtx.DB, &policy)
	if err != nil {
		return nil, err
	}

	server, err := findPrimaryCaddyServer(l.svcCtx.DB)
	if err != nil {
		return nil, err
	}

	candidateConfig, err := applyWafPolicyToCaddyConfig(server.Config, directives)
	if err != nil {
		return nil, err
	}

	if err := adaptCaddyfile(server, candidateConfig); err != nil {
		return nil, fmt.Errorf("policy validate failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
