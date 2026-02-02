package notification

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"logflux/model"

	"gorm.io/gorm"
)

func (m *Manager) workerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case jobID := <-m.workCh:
			m.processJob(ctx, jobID)
		}
	}
}

func (m *Manager) scanLoop(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.dispatchDueJobs(ctx)
		}
	}
}

func (m *Manager) dispatchDueJobs(ctx context.Context) {
	var jobs []model.NotificationJob
	err := m.db.WithContext(ctx).
		Where("status = ? AND next_run_at <= ?", model.NotificationJobStatusQueued, time.Now()).
		Order("id asc").
		Limit(100).
		Find(&jobs).Error
	if err != nil {
		m.logger.Errorf("Failed to scan notification jobs: %v", err)
		return
	}

	for _, job := range jobs {
		select {
		case m.workCh <- job.ID:
		default:
			return
		}
	}
}

func (m *Manager) processJob(ctx context.Context, jobID uint) {
	// 1) Claim: queued -> processing
	res := m.db.WithContext(ctx).
		Model(&model.NotificationJob{}).
		Where("id = ? AND status = ?", jobID, model.NotificationJobStatusQueued).
		Updates(map[string]interface{}{
			"status":          model.NotificationJobStatusProcessing,
			"last_attempt_at": time.Now(),
		})
	if res.Error != nil {
		m.logger.Errorf("Failed to claim job %d: %v", jobID, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		return
	}

	// 2) Load job
	var job model.NotificationJob
	if err := m.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		m.logger.Errorf("Failed to load job %d: %v", jobID, err)
		return
	}

	// 3) Load channel (latest config)
	var channel model.NotificationChannel
	if err := m.db.WithContext(ctx).First(&channel, job.ChannelID).Error; err != nil {
		// channel 不存在时无法重试
		m.db.WithContext(ctx).Model(&model.NotificationLog{}).
			Where("id = ?", job.LogID).
			Updates(map[string]interface{}{
				"status":        model.NotificationStatusFailed,
				"error_message": fmt.Sprintf("channel not found: %v", err),
			})
		m.db.WithContext(ctx).Model(&model.NotificationJob{}).
			Where("id = ?", job.ID).
			Updates(map[string]interface{}{
				"status":     model.NotificationJobStatusFailed,
				"last_error": fmt.Sprintf("channel not found: %v", err),
			})
		return
	}

	provider, ok := m.providers[channel.Type]
	if !ok {
		m.failJob(ctx, &job, &channel, fmt.Sprintf("provider not found: %s", channel.Type))
		return
	}

	// 4) Update log to sending
	m.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Where("id = ?", job.LogID).
		Updates(map[string]interface{}{
			"status": model.NotificationStatusSending,
		})

	// 5) Render template at execution time, using latest templates
	eventData := map[string]interface{}{}
	if job.EventData != nil {
		for k, v := range job.EventData {
			eventData[k] = v
		}
	}

	event := &Event{
		Type:      job.EventType,
		Level:     job.EventLevel,
		Title:     job.EventTitle,
		Message:   job.EventMessage,
		Data:      eventData,
		Timestamp: time.Now(),
	}

	// Always try reload templates before rendering, to reflect latest user changes.
	// This is a simple approach for single instance; can be optimized later.
	if m.templateMgr != nil {
		_ = m.templateMgr.LoadTemplates()
		if job.TemplateName != "" {
			if content, err := m.templateMgr.Render(job.TemplateName, event); err == nil {
				event.Data["rendered_content"] = content
			} else {
				m.logger.Errorf("Failed to render template %s: %v", job.TemplateName, err)
			}
		}
	}

	// 6) Send
	if err := provider.Send(ctx, map[string]interface{}(channel.Config), event); err != nil {
		m.failJob(ctx, &job, &channel, err.Error())
		return
	}

	// 6) Success: update log + job
	now := time.Now()
	m.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Where("id = ?", job.LogID).
		Updates(map[string]interface{}{
			"status":        model.NotificationStatusSuccess,
			"error_message": "",
			"sent_at":       &now,
		})

	m.db.WithContext(ctx).Model(&model.NotificationJob{}).
		Where("id = ?", job.ID).
		Updates(map[string]interface{}{
			"status": model.NotificationJobStatusSucceeded,
		})
}

type retryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Factor      float64
	Jitter      bool
}

func defaultRetryPolicy() retryPolicy {
	return retryPolicy{
		MaxAttempts: 5,
		BaseDelay:   5 * time.Second,
		MaxDelay:    10 * time.Minute,
		Factor:      2,
		Jitter:      true,
	}
}

func parseRetryPolicy(config model.JSONMap) retryPolicy {
	policy := defaultRetryPolicy()
	if config == nil {
		return policy
	}

	retryAny, ok := config["retry"]
	if !ok || retryAny == nil {
		return policy
	}

	retryMap, ok := retryAny.(map[string]interface{})
	if !ok || retryMap == nil {
		return policy
	}

	if v, ok := retryMap["maxAttempts"]; ok {
		switch n := v.(type) {
		case float64:
			policy.MaxAttempts = int(n)
		case int:
			policy.MaxAttempts = n
		}
	}
	if v, ok := retryMap["baseDelay"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil {
			policy.BaseDelay = d
		}
	}
	if v, ok := retryMap["maxDelay"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil {
			policy.MaxDelay = d
		}
	}
	if v, ok := retryMap["factor"]; ok {
		switch f := v.(type) {
		case float64:
			policy.Factor = f
		case int:
			policy.Factor = float64(f)
		}
	}
	if v, ok := retryMap["jitter"].(bool); ok {
		policy.Jitter = v
	}

	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = 5 * time.Second
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 10 * time.Minute
	}
	if policy.Factor < 1 {
		policy.Factor = 1
	}

	return policy
}

func (m *Manager) scheduleRetry(ctx context.Context, job *model.NotificationJob, channel *model.NotificationChannel, errMsg string) {
	policy := parseRetryPolicy(channel.Config)
	nextAttempt := job.RetryCount + 1

	// 达到最大次数：终态 failed
	if nextAttempt >= policy.MaxAttempts {
		m.db.WithContext(ctx).Model(&model.NotificationLog{}).
			Where("id = ?", job.LogID).
			Updates(map[string]interface{}{
				"status":        model.NotificationStatusFailed,
				"error_message": errMsg,
			})

		m.db.WithContext(ctx).Model(&model.NotificationJob{}).
			Where("id = ?", job.ID).
			Updates(map[string]interface{}{
				"status":      model.NotificationJobStatusFailed,
				"retry_count": nextAttempt,
				"last_error":  errMsg,
			})
		return
	}

	// 还有重试次数：log 回到 pending，等待下一次发送
	m.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Where("id = ?", job.LogID).
		Updates(map[string]interface{}{
			"status":        model.NotificationStatusPending,
			"error_message": errMsg,
		})

	// 计算 next_run_at（指数退避）
	delay := float64(policy.BaseDelay)
	for i := 0; i < nextAttempt; i++ {
		delay *= policy.Factor
		if time.Duration(delay) >= policy.MaxDelay {
			delay = float64(policy.MaxDelay)
			break
		}
	}

	d := time.Duration(delay)
	if d > policy.MaxDelay {
		d = policy.MaxDelay
	}
	if policy.Jitter {
		// full jitter: [0, d]
		if d > 0 {
			d = time.Duration(rand.Int63n(int64(d) + 1))
		}
	}

	nextRunAt := time.Now().Add(d)

	m.db.WithContext(ctx).Model(&model.NotificationJob{}).
		Where("id = ?", job.ID).
		Updates(map[string]interface{}{
			"status":      model.NotificationJobStatusQueued,
			"retry_count": nextAttempt,
			"next_run_at": nextRunAt,
			"last_error":  errMsg,
		})
}

func (m *Manager) failJob(ctx context.Context, job *model.NotificationJob, channel *model.NotificationChannel, errMsg string) {
	// 默认开启重试
	m.scheduleRetry(ctx, job, channel, errMsg)
}

var _ = gorm.ErrRecordNotFound
