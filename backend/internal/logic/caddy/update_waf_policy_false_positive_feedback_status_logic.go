package caddy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UpdateWafPolicyFalsePositiveFeedbackStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateWafPolicyFalsePositiveFeedbackStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWafPolicyFalsePositiveFeedbackStatusLogic {
	return &UpdateWafPolicyFalsePositiveFeedbackStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateWafPolicyFalsePositiveFeedbackStatusLogic) UpdateWafPolicyFalsePositiveFeedbackStatus(req *types.WafPolicyFalsePositiveFeedbackStatusUpdateReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil {
		return nil, fmt.Errorf("误报反馈状态更新参数不合法")
	}
	if req.ID == 0 {
		return nil, fmt.Errorf("误报反馈 ID 不能为空")
	}

	feedbackStatus := normalizePolicyFeedbackStatus(req.FeedbackStatus)
	if err := validatePolicyFeedbackStatus(feedbackStatus); err != nil {
		return nil, err
	}
	dueAt, err := parsePolicyFeedbackDueAt(req.DueAt)
	if err != nil {
		return nil, err
	}

	var feedback model.WafPolicyFalsePositiveFeedback
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&feedback, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到误报反馈记录")
		}
		return nil, fmt.Errorf("查询误报反馈失败: %w", err)
	}

	updates := map[string]interface{}{
		"feedback_status": feedbackStatus,
		"process_note":    strings.TrimSpace(req.ProcessNote),
		"assignee":        strings.TrimSpace(req.Assignee),
		"due_at":          dueAt,
	}
	if feedbackStatus == wafFeedbackStatusPending {
		updates["processed_by"] = ""
		updates["processed_at"] = nil
	} else {
		now := time.Now()
		updates["processed_by"] = currentOperatorFromContext(l.ctx)
		updates["processed_at"] = &now
	}

	if err := l.svcCtx.DB.WithContext(l.ctx).Model(&model.WafPolicyFalsePositiveFeedback{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新误报反馈状态失败: %w", err)
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "成功",
	}, nil
}
