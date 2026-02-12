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

type ListWafPolicyFalsePositiveFeedbacksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafPolicyFalsePositiveFeedbacksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafPolicyFalsePositiveFeedbacksLogic {
	return &ListWafPolicyFalsePositiveFeedbacksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafPolicyFalsePositiveFeedbacksLogic) ListWafPolicyFalsePositiveFeedbacks(req *types.WafPolicyFalsePositiveFeedbackListReq) (resp *types.WafPolicyFalsePositiveFeedbackListResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil {
		req = &types.WafPolicyFalsePositiveFeedbackListReq{}
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	db := l.svcCtx.DB.Model(&model.WafPolicyFalsePositiveFeedback{})
	if req.PolicyId > 0 {
		db = db.Where("policy_id = ?", req.PolicyId)
	}
	if host := normalizePolicyScopeHost(req.Host); host != "" {
		db = db.Where("LOWER(host) = ?", host)
	}
	if path := strings.TrimSpace(req.Path); path != "" {
		db = db.Where("path = ?", normalizePolicyScopePath(path))
	}
	if method := normalizePolicyHTTPMethod(req.Method); method != "" {
		if err := validatePolicyHTTPMethod(method); err != nil {
			return nil, err
		}
		db = db.Where("method = ?", method)
	}
	if feedbackStatus := normalizePolicyFeedbackStatus(req.FeedbackStatus); strings.TrimSpace(req.FeedbackStatus) != "" {
		if err := validatePolicyFeedbackStatus(feedbackStatus); err != nil {
			return nil, err
		}
		db = db.Where("feedback_status = ?", feedbackStatus)
	}
	if assignee := strings.TrimSpace(req.Assignee); assignee != "" {
		db = db.Where("assignee ILIKE ?", "%"+assignee+"%")
	}
	slaStatus := normalizePolicyFeedbackSLAStatus(req.SLAStatus)
	if strings.TrimSpace(req.SLAStatus) != "" {
		if err := validatePolicyFeedbackSLAStatus(slaStatus); err != nil {
			return nil, err
		}
	}
	now := time.Now()
	switch slaStatus {
	case wafFeedbackSLAStatusOverdue:
		db = db.Where("feedback_status IN ? AND due_at IS NOT NULL AND due_at < ?", []string{wafFeedbackStatusPending, wafFeedbackStatusConfirmed}, now)
	case wafFeedbackSLAStatusResolved:
		db = db.Where("feedback_status = ?", wafFeedbackStatusResolved)
	case wafFeedbackSLAStatusNormal:
		db = db.Where(
			"feedback_status IN ? AND (due_at IS NULL OR due_at >= ?)",
			[]string{wafFeedbackStatusPending, wafFeedbackStatusConfirmed}, now,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count policy false positive feedbacks failed: %w", err)
	}

	var feedbacks []model.WafPolicyFalsePositiveFeedback
	offset := (page - 1) * pageSize
	if err := db.Order("created_at desc, id desc").Limit(pageSize).Offset(offset).Find(&feedbacks).Error; err != nil {
		return nil, fmt.Errorf("query policy false positive feedbacks failed: %w", err)
	}

	policyNameMap := map[uint]string{}
	if len(feedbacks) > 0 {
		policyIDs := make([]uint, 0, len(feedbacks))
		seen := make(map[uint]struct{}, len(feedbacks))
		for _, item := range feedbacks {
			if item.PolicyID == 0 {
				continue
			}
			if _, ok := seen[item.PolicyID]; ok {
				continue
			}
			seen[item.PolicyID] = struct{}{}
			policyIDs = append(policyIDs, item.PolicyID)
		}
		if len(policyIDs) > 0 {
			var policies []model.WafPolicy
			if err := l.svcCtx.DB.Model(&model.WafPolicy{}).Where("id IN ?", policyIDs).Find(&policies).Error; err != nil {
				return nil, fmt.Errorf("query policy names failed: %w", err)
			}
			for _, policy := range policies {
				name := strings.TrimSpace(policy.Name)
				if name == "" {
					name = fmt.Sprintf("#%d", policy.ID)
				}
				policyNameMap[policy.ID] = name
			}
		}
	}

	list := make([]types.WafPolicyFalsePositiveFeedbackItem, 0, len(feedbacks))
	for _, item := range feedbacks {
		policyName := "全部策略"
		if item.PolicyID > 0 {
			policyName = policyNameMap[item.PolicyID]
			if strings.TrimSpace(policyName) == "" {
				policyName = fmt.Sprintf("#%d", item.PolicyID)
			}
		}
		statusValue := normalizePolicyFeedbackStatus(item.FeedbackStatus)
		overdue := isPolicyFeedbackOverdue(statusValue, item.DueAt, now)
		list = append(list, types.WafPolicyFalsePositiveFeedbackItem{
			ID:             item.ID,
			PolicyId:       item.PolicyID,
			PolicyName:     policyName,
			Host:           item.Host,
			Path:           item.Path,
			Method:         item.Method,
			Status:         item.Status,
			FeedbackStatus: statusValue,
			Assignee:       item.Assignee,
			DueAt:          formatNullableTime(item.DueAt),
			IsOverdue:      overdue,
			SampleURI:      item.SampleURI,
			Reason:         item.Reason,
			Suggestion:     item.Suggestion,
			Operator:       item.Operator,
			ProcessNote:    item.ProcessNote,
			ProcessedBy:    item.ProcessedBy,
			ProcessedAt:    formatNullableTime(item.ProcessedAt),
			CreatedAt:      formatTime(item.CreatedAt),
		})
	}

	return &types.WafPolicyFalsePositiveFeedbackListResp{
		List:  list,
		Total: total,
	}, nil
}
