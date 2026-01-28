package notification

import (
	"context"
	"fmt"
	"logflux/model"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// Manager 通知管理器实现
type Manager struct {
	db      *gorm.DB
	redis   *redis.Client
	logger  logx.Logger
	mu      sync.RWMutex
	started bool

	// 通知提供者 (type -> provider)
	providers map[string]NotificationProvider

	// 通知渠道 (从数据库加载)
	channels map[uint]*model.NotificationChannel

	// 告警规则 (从数据库加载)
	rules map[uint]*model.NotificationRule
}

// NewManager 创建通知管理器
func NewManager(db *gorm.DB, redis *redis.Client) *Manager {
	return &Manager{
		db:        db,
		redis:     redis,
		logger:    logx.WithContext(context.Background()),
		providers: make(map[string]NotificationProvider),
		channels:  make(map[uint]*model.NotificationChannel),
		rules:     make(map[uint]*model.NotificationRule),
	}
}

// RegisterProvider 注册通知提供者
func (m *Manager) RegisterProvider(provider NotificationProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	providerType := provider.Type()
	if _, exists := m.providers[providerType]; exists {
		return fmt.Errorf("provider %s already registered", providerType)
	}

	m.providers[providerType] = provider
	m.logger.Infof("Registered notification provider: %s", providerType)
	return nil
}

// Start 启动通知管理器
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("notification manager already started")
	}

	// 加载通知渠道
	if err := m.loadChannelsLocked(); err != nil {
		return fmt.Errorf("failed to load channels: %w", err)
	}

	// 加载告警规则
	if err := m.loadRulesLocked(); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	m.started = true
	m.logger.Info("Notification manager started")
	return nil
}

// Stop 停止通知管理器
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	m.started = false
	m.logger.Info("Notification manager stopped")
	return nil
}

// ReloadChannels 重新加载通知渠道配置
func (m *Manager) ReloadChannels() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.loadChannelsLocked()
}

// ReloadRules 重新加载告警规则
func (m *Manager) ReloadRules() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.loadRulesLocked()
}

// loadChannelsLocked 加载通知渠道 (需要持有锁)
func (m *Manager) loadChannelsLocked() error {
	var channels []model.NotificationChannel
	if err := m.db.Where("enabled = ?", true).Find(&channels).Error; err != nil {
		return err
	}

	// 清空并重新加载
	m.channels = make(map[uint]*model.NotificationChannel)
	for i := range channels {
		m.channels[channels[i].ID] = &channels[i]
	}

	m.logger.Infof("Loaded %d notification channels", len(m.channels))
	return nil
}

// loadRulesLocked 加载告警规则 (需要持有锁)
func (m *Manager) loadRulesLocked() error {
	var rules []model.NotificationRule
	if err := m.db.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return err
	}

	// 清空并重新加载
	m.rules = make(map[uint]*model.NotificationRule)
	for i := range rules {
		m.rules[rules[i].ID] = &rules[i]
	}

	m.logger.Infof("Loaded %d notification rules", len(m.rules))
	return nil
}

// Notify 发送通知
func (m *Manager) Notify(ctx context.Context, event *Event) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return fmt.Errorf("notification manager not started")
	}

	// 查找匹配的通知渠道
	matchedChannels := m.findMatchingChannels(event.Type)
	if len(matchedChannels) == 0 {
		m.logger.Infof("No matching channels for event: %s", event.Type)
		return nil
	}

	// 异步发送通知到所有匹配的渠道
	var wg sync.WaitGroup
	for _, channel := range matchedChannels {
		wg.Add(1)
		go func(ch *model.NotificationChannel) {
			defer wg.Done()
			m.sendToChannel(ctx, ch, event, nil)
		}(channel)
	}

	wg.Wait()
	return nil
}

// findMatchingChannels 查找匹配的通知渠道
func (m *Manager) findMatchingChannels(eventType string) []*model.NotificationChannel {
	var matched []*model.NotificationChannel

	for _, channel := range m.channels {
		if m.eventMatches(eventType, channel.Events) {
			matched = append(matched, channel)
		}
	}

	return matched
}

// eventMatches 检查事件类型是否匹配渠道订阅
func (m *Manager) eventMatches(eventType string, subscribedEvents []string) bool {
	for _, pattern := range subscribedEvents {
		if m.matchPattern(eventType, pattern) {
			return true
		}
	}
	return false
}

// matchPattern 匹配事件类型模式 (支持通配符 *)
func (m *Manager) matchPattern(eventType, pattern string) bool {
	// 精确匹配
	if eventType == pattern {
		return true
	}

	// 通配符匹配 (如 system.* 匹配 system.startup)
	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2]
		return strings.HasPrefix(eventType, prefix+".")
	}

	// * 匹配所有
	if pattern == "*" {
		return true
	}

	return false
}

// sendToChannel 发送通知到指定渠道
func (m *Manager) sendToChannel(ctx context.Context, channel *model.NotificationChannel, event *Event, rule *model.NotificationRule) {
	// 创建通知日志
	log := &model.NotificationLog{
		ChannelID: channel.ID,
		EventType: event.Type,
		EventData: model.JSONMap(event.Data),
		Status:    model.NotificationStatusPending,
	}
	if rule != nil {
		log.RuleID = &rule.ID
	}

	// 保存日志到数据库
	if err := m.db.Create(log).Error; err != nil {
		m.logger.Errorf("Failed to create notification log: %v", err)
		return
	}

	// 获取提供者
	provider, exists := m.providers[channel.Type]
	if !exists {
		m.updateLogStatus(log.ID, model.NotificationStatusFailed, fmt.Sprintf("Provider %s not found", channel.Type))
		m.logger.Errorf("Provider %s not found for channel %s", channel.Type, channel.Name)
		return
	}

	// 发送通知
	startTime := time.Now()
	// 注意: channel.Config 是 JSONMap 类型，需要转换为 map[string]interface{}
	err := provider.Send(ctx, map[string]interface{}(channel.Config), event)
	duration := time.Since(startTime)

	// 更新日志状态
	now := time.Now()
	if err != nil {
		log.Status = model.NotificationStatusFailed
		log.ErrorMessage = err.Error()
		m.logger.Errorf("Failed to send notification via %s: %v (took %v)", channel.Name, err, duration)
	} else {
		log.Status = model.NotificationStatusSuccess
		log.SentAt = &now
		m.logger.Infof("Successfully sent notification via %s (took %v)", channel.Name, duration)
	}

	m.db.Model(log).Updates(map[string]interface{}{
		"status":        log.Status,
		"error_message": log.ErrorMessage,
		"sent_at":       log.SentAt,
	})
}

// updateLogStatus 更新通知日志状态
func (m *Manager) updateLogStatus(logID uint, status, errorMessage string) {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}
	if status == model.NotificationStatusSuccess {
		now := time.Now()
		updates["sent_at"] = &now
	}

	m.db.Model(&model.NotificationLog{}).Where("id = ?", logID).Updates(updates)
}

// EvaluateRules 评估规则并触发通知
func (m *Manager) EvaluateRules(ctx context.Context, data map[string]interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return fmt.Errorf("notification manager not started")
	}

	// TODO: 实现规则评估逻辑
	// 这将在后续阶段实现规则引擎时完成

	return nil
}
