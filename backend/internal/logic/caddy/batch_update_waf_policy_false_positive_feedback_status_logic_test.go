package caddy

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/svc"
	"logflux/internal/types"
)

func TestBatchUpdateWafPolicyFalsePositiveFeedbackStatusSuccess(t *testing.T) {
	db, mock, cleanup := newBatchFeedbackMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "waf_policy_false_positive_feedbacks"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectExec(`UPDATE "waf_policy_false_positive_feedbacks" SET`).
		WillReturnResult(sqlmock.NewResult(0, 2))

	ctx := context.WithValue(context.Background(), "userId", "alice")
	logic := NewBatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic(ctx, &svc.ServiceContext{DB: db})
	resp, err := logic.BatchUpdateWafPolicyFalsePositiveFeedbackStatus(&types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateReq{
		IDs:            []uint{1, 2, 2, 0},
		FeedbackStatus: "confirmed",
		ProcessNote:    "bulk process",
		Assignee:       "alice",
		DueAt:          "2026-02-20 12:00:00",
	})
	if err != nil {
		t.Fatalf("BatchUpdateWafPolicyFalsePositiveFeedbackStatus() error = %v", err)
	}
	if resp == nil {
		t.Fatalf("expected non-nil response")
	}
	if resp.AffectedCount != 2 {
		t.Fatalf("expected affectedCount=2, got %d", resp.AffectedCount)
	}
	if resp.ProcessedBy != "alice" {
		t.Fatalf("expected processedBy=alice, got %q", resp.ProcessedBy)
	}
	if strings.TrimSpace(resp.ProcessedAt) == "" {
		t.Fatalf("expected processedAt not empty")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestBatchUpdateWafPolicyFalsePositiveFeedbackStatusPendingClearsAudit(t *testing.T) {
	db, mock, cleanup := newBatchFeedbackMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "waf_policy_false_positive_feedbacks"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec(`UPDATE "waf_policy_false_positive_feedbacks" SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	logic := NewBatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic(context.Background(), &svc.ServiceContext{DB: db})
	resp, err := logic.BatchUpdateWafPolicyFalsePositiveFeedbackStatus(&types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateReq{
		IDs:            []uint{9},
		FeedbackStatus: "pending",
		ProcessNote:    "reset",
	})
	if err != nil {
		t.Fatalf("BatchUpdateWafPolicyFalsePositiveFeedbackStatus() error = %v", err)
	}
	if resp == nil {
		t.Fatalf("expected non-nil response")
	}
	if resp.AffectedCount != 1 {
		t.Fatalf("expected affectedCount=1, got %d", resp.AffectedCount)
	}
	if resp.ProcessedBy != "" {
		t.Fatalf("expected processedBy empty, got %q", resp.ProcessedBy)
	}
	if resp.ProcessedAt != "" {
		t.Fatalf("expected processedAt empty, got %q", resp.ProcessedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestBatchUpdateWafPolicyFalsePositiveFeedbackStatusInvalidPayload(t *testing.T) {
	logic := NewBatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic(context.Background(), &svc.ServiceContext{})

	_, err := logic.BatchUpdateWafPolicyFalsePositiveFeedbackStatus(nil)
	if err == nil || !strings.Contains(err.Error(), "误报反馈批量状态更新参数不合法") {
		t.Fatalf("expected invalid payload error, got %v", err)
	}

	_, err = logic.BatchUpdateWafPolicyFalsePositiveFeedbackStatus(&types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateReq{
		IDs:            []uint{0, 0},
		FeedbackStatus: "confirmed",
	})
	if err == nil || !strings.Contains(err.Error(), "误报反馈 ID 列表不能为空") {
		t.Fatalf("expected ids required error, got %v", err)
	}

	tooMany := make([]uint, 0, 205)
	for i := 1; i <= 205; i++ {
		tooMany = append(tooMany, uint(i))
	}
	_, err = logic.BatchUpdateWafPolicyFalsePositiveFeedbackStatus(&types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateReq{
		IDs:            tooMany,
		FeedbackStatus: "confirmed",
	})
	if err == nil || !strings.Contains(err.Error(), "误报反馈 ID 数量超出限制") {
		t.Fatalf("expected ids limit error, got %v", err)
	}
}

func TestBatchUpdateWafPolicyFalsePositiveFeedbackStatusNotFoundAndInvalidDueAt(t *testing.T) {
	db, mock, cleanup := newBatchFeedbackMockDB(t)
	defer cleanup()

	logic := NewBatchUpdateWafPolicyFalsePositiveFeedbackStatusLogic(context.Background(), &svc.ServiceContext{DB: db})
	_, err := logic.BatchUpdateWafPolicyFalsePositiveFeedbackStatus(&types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateReq{
		IDs:            []uint{1},
		FeedbackStatus: "resolved",
		DueAt:          "not-a-time",
	})
	if err == nil || !strings.Contains(err.Error(), "截止时间格式不合法") {
		t.Fatalf("expected invalid dueAt error, got %v", err)
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "waf_policy_false_positive_feedbacks"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	_, err = logic.BatchUpdateWafPolicyFalsePositiveFeedbackStatus(&types.WafPolicyFalsePositiveFeedbackBatchStatusUpdateReq{
		IDs:            []uint{1},
		FeedbackStatus: "resolved",
	})
	if err == nil || !strings.Contains(err.Error(), "未找到误报反馈记录") {
		t.Fatalf("expected not found error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func newBatchFeedbackMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		_ = sqldb.Close()
		t.Fatalf("failed to open gorm: %v", err)
	}

	cleanup := func() {
		_ = sqldb.Close()
	}
	return gdb, mock, cleanup
}
