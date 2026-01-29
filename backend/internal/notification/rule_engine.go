package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"logflux/model"
	"regexp"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/redis/go-redis/v9"
)

// RuleEngine 规则引擎接口
type RuleEngine interface {
	// Evaluate 评估规则是否触发
	Evaluate(ctx context.Context, rule *model.NotificationRule, event *Event) (bool, error)

	// GetMatchingRules 获取匹配事件的所有规则
	GetMatchingRules(ctx context.Context, eventType string) ([]*model.NotificationRule, error)
}

// ruleEngine 规则引擎实现
type ruleEngine struct {
	redis    *redis.Client
	evaluators map[string]RuleEvaluator
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine(redis *redis.Client) RuleEngine {
	engine := &ruleEngine{
		redis:    redis,
		evaluators: make(map[string]RuleEvaluator),
	}

	// 注册内置评估器
	engine.RegisterEvaluator(model.RuleTypeThreshold, NewThresholdEvaluator())
	engine.RegisterEvaluator(model.RuleTypeFrequency, NewFrequencyEvaluator(redis))
	engine.RegisterEvaluator(model.RuleTypePattern, NewPatternEvaluator())

	return engine
}

// RegisterEvaluator 注册规则评估器
func (e *ruleEngine) RegisterEvaluator(ruleType string, evaluator RuleEvaluator) {
	e.evaluators[ruleType] = evaluator
}

// Evaluate 评估规则是否触发
func (e *ruleEngine) Evaluate(ctx context.Context, rule *model.NotificationRule, event *Event) (bool, error) {
	// 检查规则是否启用
	if !rule.Enabled {
		return false, nil
	}

	// 检查事件类型是否匹配
	if rule.EventType != "*" && !matchEventType(rule.EventType, event.Type) {
		return false, nil
	}

	// 检查静默期
	if rule.LastTriggeredAt != nil {
		silenceDuration := time.Duration(rule.SilenceDuration) * time.Second
		if time.Since(*rule.LastTriggeredAt) < silenceDuration {
			return false, nil
		}
	}

	// 根据规则类型选择评估器
	evaluator, exists := e.evaluators[rule.RuleType]
	if !exists {
		return false, fmt.Errorf("unsupported rule type: %s", rule.RuleType)
	}

	// 评估规则
	return evaluator.Evaluate(ctx, rule.Condition, event)
}

// GetMatchingRules 获取匹配事件的所有规则
func (e *ruleEngine) GetMatchingRules(ctx context.Context, eventType string) ([]*model.NotificationRule, error) {
	// 这个方法将由 Manager 实现,因为它需要访问数据库
	// 这里只是接口定义
	return nil, nil
}

// RuleEvaluator 规则评估器接口
type RuleEvaluator interface {
	Evaluate(ctx context.Context, condition model.JSONMap, event *Event) (bool, error)
}

// ThresholdEvaluator 阈值规则评估器
type ThresholdEvaluator struct {
	cache map[string]*vm.Program // 缓存编译后的表达式
}

// NewThresholdEvaluator 创建阈值评估器
func NewThresholdEvaluator() *ThresholdEvaluator {
	return &ThresholdEvaluator{
		cache: make(map[string]*vm.Program),
	}
}

// Evaluate 评估阈值规则
func (t *ThresholdEvaluator) Evaluate(ctx context.Context, condition model.JSONMap, event *Event) (bool, error) {
	// 解析条件
	var cond model.ThresholdCondition
	if err := mapToCondition(condition, &cond); err != nil {
		return false, fmt.Errorf("invalid threshold condition: %w", err)
	}

	// 构建表达式
	expression := fmt.Sprintf("data.%s %s value", cond.Field, cond.Operator)

	// 编译表达式 (使用缓存)
	program, exists := t.cache[expression]
	if !exists {
		compiled, err := expr.Compile(expression, expr.Env(map[string]interface{}{
			"data":  event.Data,
			"value": cond.Value,
		}))
		if err != nil {
			return false, fmt.Errorf("failed to compile expression: %w", err)
		}
		program = compiled
		t.cache[expression] = program
	}

	// 执行表达式
	env := map[string]interface{}{
		"data":  event.Data,
		"value": cond.Value,
	}
	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	// 转换结果为布尔值
	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("expression result is not boolean: %T", output)
	}

	return result, nil
}

// FrequencyEvaluator 频率规则评估器
type FrequencyEvaluator struct {
	redis *redis.Client
}

// NewFrequencyEvaluator 创建频率评估器
func NewFrequencyEvaluator(redis *redis.Client) *FrequencyEvaluator {
	return &FrequencyEvaluator{
		redis: redis,
	}
}

// Evaluate 评估频率规则
func (f *FrequencyEvaluator) Evaluate(ctx context.Context, condition model.JSONMap, event *Event) (bool, error) {
	// 解析条件
	var cond model.FrequencyCondition
	if err := mapToCondition(condition, &cond); err != nil {
		return false, fmt.Errorf("invalid frequency condition: %w", err)
	}

	// 解析时间窗口
	window, err := time.ParseDuration(cond.Window)
	if err != nil {
		return false, fmt.Errorf("invalid window format: %w", err)
	}

	// 构建 Redis 键
	var key string
	if cond.GroupBy != "" {
		// 分组统计
		groupValue, ok := event.Data[cond.GroupBy]
		if !ok {
			return false, nil // 没有分组字段,不触发
		}
		key = fmt.Sprintf("rule:frequency:%s:%s:%v", cond.Event, cond.GroupBy, groupValue)
	} else {
		// 全局统计
		key = fmt.Sprintf("rule:frequency:%s:global", cond.Event)
	}

	// 增加计数
	count, err := f.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment counter: %w", err)
	}

	// 设置过期时间 (仅第一次)
	if count == 1 {
		f.redis.Expire(ctx, key, window)
	}

	// 检查是否超过阈值
	return count >= int64(cond.Count), nil
}

// PatternEvaluator 模式匹配规则评估器
type PatternEvaluator struct {
	cache map[string]*regexp.Regexp // 缓存编译后的正则表达式
}

// NewPatternEvaluator 创建模式评估器
func NewPatternEvaluator() *PatternEvaluator {
	return &PatternEvaluator{
		cache: make(map[string]*regexp.Regexp),
	}
}

// Evaluate 评估模式匹配规则
func (p *PatternEvaluator) Evaluate(ctx context.Context, condition model.JSONMap, event *Event) (bool, error) {
	// 解析条件
	var cond model.PatternCondition
	if err := mapToCondition(condition, &cond); err != nil {
		return false, fmt.Errorf("invalid pattern condition: %w", err)
	}

	// 获取字段值
	fieldValue, ok := event.Data[cond.Field]
	if !ok {
		return false, nil // 字段不存在,不触发
	}

	// 转换为字符串
	strValue := fmt.Sprintf("%v", fieldValue)

	// 编译正则表达式 (使用缓存)
	regex, exists := p.cache[cond.Pattern]
	if !exists {
		compiled, err := regexp.Compile(cond.Pattern)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern: %w", err)
		}
		regex = compiled
		p.cache[cond.Pattern] = regex
	}

	// 匹配
	return regex.MatchString(strValue), nil
}

// 辅助函数

// mapToCondition 将 JSONMap 转换为条件结构体
func mapToCondition(m model.JSONMap, v interface{}) error {
	// 使用 JSON 序列化/反序列化转换
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// matchEventType 匹配事件类型 (支持通配符)
func matchEventType(pattern, eventType string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == eventType {
		return true
	}
	// 支持前缀通配符,如 "system.*"
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix
	}
	return false
}
