package notification

import (
	"context"
	"fmt"
	"logflux/internal/notification/template"
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

	// 规则引擎
	ruleEngine RuleEngine

	// 模板管理器
	templateMgr *template.TemplateManager

	// async dispatch
	workCh chan uint
}

// NewManager 创建通知管理器
func NewManager(db *gorm.DB, redis *redis.Client, tm *template.TemplateManager) *Manager {
	return &Manager{
		db:          db,
		redis:       redis,
		logger:      logx.WithContext(context.Background()),
		providers:   make(map[string]NotificationProvider),
		channels:    make(map[uint]*model.NotificationChannel),
		rules:       make(map[uint]*model.NotificationRule),
		ruleEngine:  NewRuleEngine(redis),
		templateMgr: tm,
		workCh:      make(chan uint, 1024),
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

	// 启动 worker pool（单实例固定 4 个 worker）
	for i := 0; i < 4; i++ {
		go m.workerLoop(ctx)
	}
	go m.scanLoop(ctx)

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

// ReloadTemplates 重新加载通知模板
func (m *Manager) ReloadTemplates() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.templateMgr.LoadTemplates()
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
	if !m.started {
		m.mu.RUnlock()
		return fmt.Errorf("notification manager not started")
	}

	// 1. 评估告警规则
	triggeredRules := m.evaluateRules(ctx, event)

	// 2. 从规则中收集渠道 ID
	ruleChannelIDs := make(map[uint]bool)
	for _, rule := range triggeredRules {
		for _, channelID := range rule.ChannelIDs {
			ruleChannelIDs[uint(channelID)] = true
		}
	}

	// 3. 查找直接匹配事件的通知渠道
	matchedChannels := m.findMatchingChannels(event.Type)

	// 4. 合并规则触发的渠道和直接匹配的渠道
	channelsToNotify := make(map[uint]*model.NotificationChannel)
	for _, channel := range matchedChannels {
		channelsToNotify[channel.ID] = channel
	}
	for channelID := range ruleChannelIDs {
		if channel, exists := m.channels[channelID]; exists {
			channelsToNotify[channelID] = channel
		}
	}
	m.mu.RUnlock()

	// 5. 更新规则触发状态（需要写锁，避免在持有读锁时升级造成死锁）
	for _, rule := range triggeredRules {
		m.updateRuleTriggerStatus(ctx, rule)
	}

	if len(channelsToNotify) == 0 {
		m.logger.Infof("No matching channels for event: %s", event.Type)
		return nil
	}

	// 6. 入队（写 notification_logs + notification_jobs），不做网络发送
	for _, channel := range channelsToNotify {
		// 查找对应的规则 (用于模板渲染)
		var rule *model.NotificationRule
		for _, r := range triggeredRules {
			for _, cid := range r.ChannelIDs {
				if uint(cid) == channel.ID {
					rule = r
					break
				}
			}
		}

		jobID := m.enqueueJob(ctx, channel, event, rule)
		if jobID > 0 {
			// 尝试低延迟派发；队列满时依赖扫描器补投递
			select {
			case m.workCh <- jobID:
			default:
			}
		}
	}

	return nil
}

// evaluateRules 评估所有规则
func (m *Manager) evaluateRules(ctx context.Context, event *Event) []*model.NotificationRule {
	var triggered []*model.NotificationRule

	for _, rule := range m.rules {
		if !rule.Enabled {
			continue
		}

		// 使用规则引擎评估
		match, err := m.ruleEngine.Evaluate(ctx, rule, event)
		if err != nil {
			m.logger.Errorf("Failed to evaluate rule %s: %v", rule.Name, err)
			continue
		}

		if match {
			triggered = append(triggered, rule)
			m.logger.Infof("Rule triggered: %s for event %s", rule.Name, event.Type)
		}
	}

	return triggered
}

// updateRuleTriggerStatus 更新规则触发状态
func (m *Manager) updateRuleTriggerStatus(ctx context.Context, rule *model.NotificationRule) {
	now := time.Now()
	updates := map[string]interface{}{
		"last_triggered_at": now,
		"trigger_count":     gorm.Expr("trigger_count + 1"),
	}

	if err := m.db.WithContext(ctx).Model(&model.NotificationRule{}).
		Where("id = ?", rule.ID).
		Updates(updates).Error; err != nil {
		m.logger.Errorf("Failed to update rule trigger status: %v", err)
	}

	// 同步更新内存中的规则状态，保证 SilenceDuration 生效。
	// 注意：Notify 期间持有 m.mu.RLock，因此这里使用 RLock 安全写入会造成数据竞争；
	// 我们在内部升级为写锁以保护 m.rules map 及 rule 指针的写。
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rules[rule.ID]; ok {
		r.LastTriggeredAt = &now
		r.TriggerCount++
	}
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
		ChannelID: &channel.ID,
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

	// 渲染通知内容
	templateName := m.determineTemplateName(channel, rule)
	if content, errRender := m.templateMgr.Render(templateName, event); errRender == nil {
		// 将渲染后的内容存入 Event Data，供 Provider 使用
		if event.Data == nil {
			event.Data = make(map[string]interface{})
		}
		event.Data["rendered_content"] = content
	} else {
		m.logger.Errorf("Failed to render template %s: %v", templateName, errRender)
		// Fallback: Provider will use event.Message
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

// determineTemplateName 确定使用的模板名称
func (m *Manager) determineTemplateName(channel *model.NotificationChannel, rule *model.NotificationRule) string {
	// 1. 优先使用规则中定义的模板
	if rule != nil && rule.Template != "" {
		return rule.Template
	}

	// 2. 其次使用系统默认的对应类型模板
	switch channel.Type {
	case "email":
		return "default_email"
	case "telegram":
		return "default_markdown"
	case "webhook":
		return "default_markdown"
	}

	return "default_markdown" // Fallback
}

func (m *Manager) enqueueJob(ctx context.Context, channel *model.NotificationChannel, event *Event, rule *model.NotificationRule) uint {
	// 渲染通知内容（在入队时渲染，避免 worker 发送时再依赖共享 event.Data）
	templateName := m.determineTemplateName(channel, rule)
	content := ""
	if m.templateMgr != nil {
		if rendered, err := m.templateMgr.Render(templateName, event); err == nil {
			content = rendered
		} else {
			m.logger.Errorf("Failed to render template %s: %v", templateName, err)
		}
	}

	// 复制一份 event data，避免并发写 map
	eventData := map[string]interface{}{}
	if event.Data != nil {
		for k, v := range event.Data {
			eventData[k] = v
		}
	}
	if eventData == nil {
		eventData = map[string]interface{}{}
	}
	// 标准字段，供 UI 展示
	eventData["title"] = event.Title
	eventData["message"] = event.Message
	eventData["level"] = event.Level
	if content != "" {
		eventData["rendered_content"] = content
	}

	// 创建通知日志
	log := &model.NotificationLog{
		ChannelID: &channel.ID,
		EventType: event.Type,
		EventData: model.JSONMap(eventData),
		Status:    model.NotificationStatusPending,
	}
	if rule != nil {
		log.RuleID = &rule.ID
	}
	if err := m.db.WithContext(ctx).Create(log).Error; err != nil {
		m.logger.Errorf("Failed to create notification log: %v", err)
		return 0
	}

	// 创建 job
	job := &model.NotificationJob{
		LogID:        log.ID,
		ChannelID:    channel.ID,
		ProviderType: channel.Type,
		EventType:    event.Type,
		EventLevel:   event.Level,
		EventTitle:   event.Title,
		EventMessage: event.Message,
		EventData:    model.JSONMap(eventData),
		TemplateName: templateName,
		Status:       model.NotificationJobStatusQueued,
		RetryCount:   0,
		NextRunAt:    time.Now(),
	}
	if err := m.db.WithContext(ctx).Create(job).Error; err != nil {
		m.logger.Errorf("Failed to create notification job: %v", err)
		// 同步标记 log 失败（避免 UI 长期 pending）
		m.updateLogStatus(log.ID, model.NotificationStatusFailed, fmt.Sprintf("enqueue failed: %v", err))
		return 0
	}

	return job.ID
}
