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
		return nil, fmt.Errorf("invalid policy false positive feedback update payload")
	}
	if req.ID == 0 {
		return nil, fmt.Errorf("policy false positive feedback id is required")
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
	if err := l.svcCtx.DB.First(&feedback, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("policy false positive feedback not found")
		}
		return nil, fmt.Errorf("query policy false positive feedback failed: %w", err)
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

	if err := l.svcCtx.DB.Model(&model.WafPolicyFalsePositiveFeedback{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update policy false positive feedback status failed: %w", err)
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
