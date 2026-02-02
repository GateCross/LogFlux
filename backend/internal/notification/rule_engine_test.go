package notification

import (
	"context"
	"logflux/model"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestThresholdEvaluator(t *testing.T) {
	evaluator := NewThresholdEvaluator()

	tests := []struct {
		name      string
		condition model.JSONMap
		event     *Event
		want      bool
		wantErr   bool
	}{
		{
			name: "greater than - true",
			condition: model.JSONMap{
				"field":    "count",
				"operator": ">",
				"value":    10.0,
			},
			event: &Event{
				Type: "test",
				Data: map[string]interface{}{
					"count": 15.0,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "greater than - false",
			condition: model.JSONMap{
				"field":    "count",
				"operator": ">",
				"value":    10.0,
			},
			event: &Event{
				Type: "test",
				Data: map[string]interface{}{
					"count": 5.0,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "equals - true",
			condition: model.JSONMap{
				"field":    "status",
				"operator": "==",
				"value":    "error",
			},
			event: &Event{
				Type: "test",
				Data: map[string]interface{}{
					"status": "error",
				},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(context.Background(), tt.condition, tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPatternEvaluator(t *testing.T) {
	evaluator := NewPatternEvaluator()

	tests := []struct {
		name      string
		condition model.JSONMap
		event     *Event
		want      bool
		wantErr   bool
	}{
		{
			name: "regex match - true",
			condition: model.JSONMap{
				"field":   "message",
				"pattern": "error.*occurred",
			},
			event: &Event{
				Type: "test",
				Data: map[string]interface{}{
					"message": "error has occurred",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "regex match - false",
			condition: model.JSONMap{
				"field":   "message",
				"pattern": "error.*occurred",
			},
			event: &Event{
				Type: "test",
				Data: map[string]interface{}{
					"message": "everything is fine",
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(context.Background(), tt.condition, tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchEventType(t *testing.T) {
	tests := []struct {
		pattern   string
		eventType string
		want      bool
	}{
		{"*", "system.startup", true},
		{"system.*", "system.startup", true},
		{"system.*", "system.shutdown", true},
		{"system.*", "error.database", false},
		{"system.startup", "system.startup", true},
		{"system.startup", "system.shutdown", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.eventType, func(t *testing.T) {
			if got := matchEventType(tt.pattern, tt.eventType); got != tt.want {
				t.Errorf("matchEventType(%v, %v) = %v, want %v", tt.pattern, tt.eventType, got, tt.want)
			}
		})
	}
}

func TestRuleEngine_Evaluate_SilencePeriod(t *testing.T) {
	// 仅测试静默期逻辑; 该测试不应依赖外部 Redis
	engine := NewRuleEngine(nil)

	// 创建规则
	lastTriggered := time.Now().Add(-1 * time.Minute) // 1 分钟前触发
	rule := &model.NotificationRule{
		ID:              1,
		Name:            "test-rule",
		Enabled:         true,
		RuleType:        model.RuleTypeThreshold,
		EventType:       "test.*",
		SilenceDuration: 300, // 5 分钟静默期
		LastTriggeredAt: &lastTriggered,
		Condition: model.JSONMap{
			"field":    "count",
			"operator": ">",
			"value":    10.0,
		},
	}

	event := &Event{
		Type: "test.event",
		Data: map[string]interface{}{
			"count": 15.0,
		},
	}

	// 应该因为静默期而不触发
	triggered, err := engine.Evaluate(context.Background(), rule, event)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if triggered {
		t.Error("Rule should not trigger during silence period")
	}
}

func TestFrequencyEvaluator_NilRedis_DoesNotPanic(t *testing.T) {
	evaluator := NewFrequencyEvaluator(nil)

	// 频率规则依赖 Redis；当 Redis 未初始化时不应 panic
	cond := model.JSONMap{
		"event":  "test.event",
		"count":  1,
		"window": "1m",
	}
	event := &Event{Type: "test.event", Data: map[string]interface{}{"foo": "bar"}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Evaluate() panicked: %v", r)
		}
	}()

	_, _ = evaluator.Evaluate(context.Background(), cond, event)
}

func TestRuleEvaluators_ConcurrentEvaluate_NoPanic(t *testing.T) {
	// 该测试主要用于捕获 concurrent map writes；它应在 -race 下跑更有效
	threshold := NewThresholdEvaluator()
	pattern := NewPatternEvaluator()

	tEvent := &Event{Type: "test", Data: map[string]interface{}{"count": 15.0, "message": "error has occurred"}}
	tCond := model.JSONMap{"field": "count", "operator": ">", "value": 10.0}
	pCond := model.JSONMap{"field": "message", "pattern": "error.*occurred"}

	const goroutines = 64
	const iterations = 200

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("concurrent Evaluate panicked: %v", r)
		}
	}()

	done := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				_, _ = threshold.Evaluate(context.Background(), tCond, tEvent)
				_, _ = pattern.Evaluate(context.Background(), pCond, tEvent)
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}

func TestManager_UpdateRuleTriggerStatus_UpdatesInMemoryRule(t *testing.T) {
	// Setup mock DB (updateRuleTriggerStatus 会写 DB)
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer sqldb.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	// 允许任意 UPDATE 语句（此处仅关心内存状态更新）
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	m := &Manager{db: gdb}
	now := time.Now().Add(-1 * time.Hour)
	rule := &model.NotificationRule{ID: 1, LastTriggeredAt: &now}
	m.rules = map[uint]*model.NotificationRule{1: rule}

	m.updateRuleTriggerStatus(context.Background(), rule)

	updated := m.rules[1]
	if updated == nil || updated.LastTriggeredAt == nil {
		t.Fatalf("expected rule.LastTriggeredAt to be updated in memory")
	}
	if !updated.LastTriggeredAt.After(now) {
		t.Fatalf("expected LastTriggeredAt to move forward, got=%v, old=%v", *updated.LastTriggeredAt, now)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
