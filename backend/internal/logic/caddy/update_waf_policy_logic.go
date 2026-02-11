package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UpdateWafPolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateWafPolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWafPolicyLogic {
	return &UpdateWafPolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateWafPolicyLogic) UpdateWafPolicy(req *types.WafPolicyUpdateReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)
	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}

	var policy model.WafPolicy
	if err := helper.svcCtx.DB.First(&policy, req.ID).Error; err != nil {
		return nil, fmt.Errorf("policy not found")
	}

	originalName := strings.TrimSpace(policy.Name)
	if err := applyPolicyUpdateReqToModel(helper, req, &policy); err != nil {
		return nil, err
	}

	if name := strings.TrimSpace(policy.Name); name == "" {
		return nil, fmt.Errorf("policy name is required")
	} else if name != originalName {
		var count int64
		if err := helper.svcCtx.DB.Model(&model.WafPolicy{}).
			Where("name = ? AND id <> ?", name, policy.ID).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("check policy name failed: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("policy name already exists: %s", name)
		}
	}

	directives, err := buildWafPolicyDirectives(&policy)
	if err != nil {
		return nil, err
	}

	operator := helper.currentOperator()
	if err := helper.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureSingleDefaultPolicy(tx, &policy); err != nil {
			return err
		}

		if err := tx.Save(&policy).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return fmt.Errorf("policy name already exists: %s", policy.Name)
			}
			return fmt.Errorf("update policy failed: %w", err)
		}

		if _, err := createPolicyRevision(tx, &policy, wafPolicyStatusDraft, directives, "update policy", operator); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
