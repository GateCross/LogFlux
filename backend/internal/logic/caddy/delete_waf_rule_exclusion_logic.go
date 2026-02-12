package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteWafRuleExclusionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteWafRuleExclusionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteWafRuleExclusionLogic {
	return &DeleteWafRuleExclusionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteWafRuleExclusionLogic) DeleteWafRuleExclusion(req *types.IDReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy exclusion id is required")
	}

	result := l.svcCtx.DB.Where("id = ?", req.ID).Delete(&model.WafRuleExclusion{})
	if result.Error != nil {
		return nil, fmt.Errorf("delete policy exclusion failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("policy exclusion not found")
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
