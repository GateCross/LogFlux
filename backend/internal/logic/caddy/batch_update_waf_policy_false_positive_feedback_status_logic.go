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
)

type BatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic {
	return &BatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic) BatchUpdateWafPolicyFalsePositiveFeedbackStatus(req *types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateReq) (resp *types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil {
		return nil, fmt.Errorf("误报反馈批量状态更新参数不合法")
	}
	feedbackIDs := normalizePolicyFeedbackIDs(req.IDs)
	if len(feedbackIDs) == 0 {
		return nil, fmt.Errorf("误报反馈 ID 列表不能为空")
	}
	if len(feedbackIDs) > 200 {
		return nil, fmt.Errorf("误报反馈 ID 数量超出限制: 200")
	}

	feedbackStatus := normalizePolicyFeedbackStatus(req.FeedbackStatus)
	if err := validatePolicyFeedbackStatus(feedbackStatus); err != nil {
		return nil, err
	}
	dueAt, err := parsePolicyFeedbackDueAt(req.DueAt)
	if err != nil {
		return nil, err
	}

	var existingCount int64
	if err := l.svcCtx.DB.WithContext(l.ctx).Model(&model.WafPolicyFalsePositiveFeedback{}).Where("id IN ?", feedbackIDs).Count(&existingCount).Error; err != nil {
		return nil, fmt.Errorf("统计误报反馈失败: %w", err)
	}
	if existingCount == 0 {
		return nil, fmt.Errorf("未找到误报反馈记录")
	}

	updates := map[string]interface{}{
		"feedback_status": feedbackStatus,
		"process_note":    strings.TrimSpace(req.ProcessNote),
		"assignee":        strings.TrimSpace(req.Assignee),
		"due_at":          dueAt,
	}
	processedBy := ""
	processedAt := ""
	if feedbackStatus == wafFeedbackStatusPending {
		updates["processed_by"] = ""
		updates["processed_at"] = nil
	} else {
		now := time.Now()
		processedBy = currentOperatorFromContext(l.ctx)
		processedAt = formatTime(now)
		updates["processed_by"] = processedBy
		updates["processed_at"] = &now
	}

	tx := l.svcCtx.DB.WithContext(l.ctx).Model(&model.WafPolicyFalsePositiveFeedback{}).Where("id IN ?", feedbackIDs).Updates(updates)
	if tx.Error != nil {
		return nil, fmt.Errorf("批量更新误报反馈状态失败: %w", tx.Error)
	}

	return &types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateResp{
		AffectedCount: tx.RowsAffected,
		ProcessedBy:   processedBy,
		ProcessedAt:   processedAt,
	}, nil
}
