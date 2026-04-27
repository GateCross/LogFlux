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
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)
	if req == nil {
		return nil, fmt.Errorf("策略参数不合法")
	}

	policy := &model.WafPolicy{}
	if err := applyPolicyReqToModel(helper, req, policy); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(policy.Name)
	if name == "" {
		return nil, fmt.Errorf("策略名称不能为空")
	}

	var existing model.WafPolicy
	if err := helper.svcCtx.DB.WithContext(helper.ctx).Where("name = ?", name).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("策略名称已存在: %s", name)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查策略名称失败: %w", err)
	}

	directives, err := buildWafPolicyDirectives(policy)
	if err != nil {
		return nil, err
	}

	operator := helper.currentOperator()
	if err := helper.svcCtx.DB.WithContext(helper.ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureSingleDefaultPolicy(tx, policy); err != nil {
			return err
		}

		if err := tx.Create(policy).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return fmt.Errorf("策略名称已存在: %s", name)
			}
			return fmt.Errorf("创建策略失败: %w", err)
		}

		if _, err := createPolicyRevision(tx, policy, wafPolicyStatusDraft, directives, "create policy", operator); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &types.BaseResp{Code: 200, Msg: "成功"}, nil
}
