package notification

import (
	"context"
	"logflux/model"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
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
	// 使用 miniredis 或 mock Redis
	// 这里简化测试,仅测试静默期逻辑
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	engine := NewRuleEngine(rdb)

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
