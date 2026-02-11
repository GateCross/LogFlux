package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewWafPolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewWafPolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewWafPolicyLogic {
	return &PreviewWafPolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewWafPolicyLogic) PreviewWafPolicy(req *types.WafPolicyActionReq) (resp *types.WafPolicyPreviewResp, err error) {
	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}

	var policy model.WafPolicy
	if err := l.svcCtx.DB.First(&policy, req.ID).Error; err != nil {
		return nil, fmt.Errorf("policy not found")
	}

	directives, err := buildWafPolicyDirectives(&policy)
	if err != nil {
		return nil, err
	}

	return &types.WafPolicyPreviewResp{Directives: directives}, nil
}
