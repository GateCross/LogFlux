package caddy

import (
	"fmt"
	"strings"
	"time"

	"logflux/internal/utils"
)

const (
	wafFeedbackStatusPending   = "pending"
	wafFeedbackStatusConfirmed = "confirmed"
	wafFeedbackStatusResolved  = "resolved"

	wafFeedbackSLAStatusAll      = "all"
	wafFeedbackSLAStatusNormal   = "normal"
	wafFeedbackSLAStatusOverdue  = "overdue"
	wafFeedbackSLAStatusResolved = "resolved"
)

func normalizePolicyFeedbackStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" {
		return wafFeedbackStatusPending
	}
	return normalized
}

func validatePolicyFeedbackStatus(status string) error {
	switch normalizePolicyFeedbackStatus(status) {
	case wafFeedbackStatusPending, wafFeedbackStatusConfirmed, wafFeedbackStatusResolved:
		return nil
	default:
		return fmt.Errorf("误报反馈状态无效: %s", status)
	}
}

func normalizePolicyFeedbackSLAStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" {
		return wafFeedbackSLAStatusAll
	}
	return normalized
}

func validatePolicyFeedbackSLAStatus(status string) error {
	switch normalizePolicyFeedbackSLAStatus(status) {
	case wafFeedbackSLAStatusAll, wafFeedbackSLAStatusNormal, wafFeedbackSLAStatusOverdue, wafFeedbackSLAStatusResolved:
		return nil
	default:
		return fmt.Errorf("误报反馈 SLA 状态无效: %s", status)
	}
}

func parsePolicyFeedbackDueAt(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	value, err := utils.ParseOptionalTime(trimmed)
	if err != nil {
		return nil, fmt.Errorf("截止时间格式不合法: %w", err)
	}
	return value, nil
}

func isPolicyFeedbackOverdue(feedbackStatus string, dueAt *time.Time, now time.Time) bool {
	status := normalizePolicyFeedbackStatus(feedbackStatus)
	if status == wafFeedbackStatusResolved {
		return false
	}
	if dueAt == nil || dueAt.IsZero() {
		return false
	}
	return dueAt.Before(now)
}

func normalizePolicyFeedbackIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
