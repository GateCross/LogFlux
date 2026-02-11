package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DeleteWafPolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteWafPolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteWafPolicyLogic {
	return &DeleteWafPolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteWafPolicyLogic) DeleteWafPolicy(req *types.IDReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)
	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}

	var policy model.WafPolicy
	if err := helper.svcCtx.DB.First(&policy, req.ID).Error; err != nil {
		return nil, fmt.Errorf("policy not found")
	}

	if err := helper.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("policy_id = ?", policy.ID).Delete(&model.WafPolicyRevision{}).Error; err != nil {
			return fmt.Errorf("delete policy revisions failed: %w", err)
		}

		if err := tx.Delete(&policy).Error; err != nil {
			return fmt.Errorf("delete policy failed: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
