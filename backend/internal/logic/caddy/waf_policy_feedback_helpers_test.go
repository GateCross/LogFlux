package caddy

import (
	"testing"
	"time"
)

func TestNormalizePolicyFeedbackIDs(t *testing.T) {
	normalized := normalizePolicyFeedbackIDs([]uint{0, 3, 2, 3, 1, 0, 2})
	if len(normalized) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(normalized))
	}
	expected := []uint{3, 2, 1}
	for index, id := range expected {
		if normalized[index] != id {
			t.Fatalf("expected normalized[%d]=%d, got %d", index, id, normalized[index])
		}
	}
}

func TestIsPolicyFeedbackOverdue(t *testing.T) {
	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.Local)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	if !isPolicyFeedbackOverdue("pending", &past, now) {
		t.Fatalf("pending with past dueAt should be overdue")
	}
	if isPolicyFeedbackOverdue("resolved", &past, now) {
		t.Fatalf("resolved should not be overdue")
	}
	if isPolicyFeedbackOverdue("confirmed", &future, now) {
		t.Fatalf("future dueAt should not be overdue")
	}
	if isPolicyFeedbackOverdue("pending", nil, now) {
		t.Fatalf("nil dueAt should not be overdue")
	}
}
