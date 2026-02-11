package caddy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CreateWafPolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateWafPolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateWafPolicyLogic {
	return &CreateWafPolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateWafPolicyLogic) CreateWafPolicy(req *types.WafPolicyReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)
	if req == nil {
		return nil, fmt.Errorf("invalid policy payload")
	}

	policy := &model.WafPolicy{}
	if err := applyPolicyReqToModel(helper, req, policy); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(policy.Name)
	if name == "" {
		return nil, fmt.Errorf("policy name is required")
	}

	var existing model.WafPolicy
	if err := helper.svcCtx.DB.Where("name = ?", name).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("policy name already exists: %s", name)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check policy name failed: %w", err)
	}

	directives, err := buildWafPolicyDirectives(policy)
	if err != nil {
		return nil, err
	}

	operator := helper.currentOperator()
	if err := helper.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureSingleDefaultPolicy(tx, policy); err != nil {
			return err
		}

		if err := tx.Create(policy).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return fmt.Errorf("policy name already exists: %s", name)
			}
			return fmt.Errorf("create policy failed: %w", err)
		}

		if _, err := createPolicyRevision(tx, policy, wafPolicyStatusDraft, directives, "create policy", operator); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
