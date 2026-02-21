package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"logflux/model"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// WafSourceJobExecutor 定义 WAF 更新源调度执行器。
type WafSourceJobExecutor interface {
	CheckSource(ctx context.Context, sourceID uint) error
	SyncSource(ctx context.Context, sourceID uint, activateNow bool) error
}

// WafScheduler 负责按 waf_sources.schedule 调度检查/同步任务。
type WafScheduler struct {
	cron *cron.Cron
	db   *gorm.DB

	mu       sync.RWMutex
	executor WafSourceJobExecutor
	started  bool
	entryMap sync.Map // map[uint]cron.EntryID
}

func NewWafScheduler(db *gorm.DB) *WafScheduler {
	return &WafScheduler{
		cron: cron.New(cron.WithSeconds()),
		db:   db,
	}
}

func (scheduler *WafScheduler) SetExecutor(executor WafSourceJobExecutor) {
	if scheduler == nil {
		return
	}

	scheduler.mu.Lock()
	scheduler.executor = executor
	started := scheduler.started
	scheduler.mu.Unlock()

	if started {
		if err := scheduler.Reload(); err != nil {
			logx.Errorf("reload waf scheduler after setting executor failed: %v", err)
		}
	}
}

func (scheduler *WafScheduler) Start() {
	if scheduler == nil {
		return
	}

	scheduler.mu.Lock()
	if scheduler.started {
		scheduler.mu.Unlock()
		return
	}
	scheduler.started = true
	scheduler.mu.Unlock()

	if err := scheduler.Reload(); err != nil {
		logx.Errorf("initial reload waf scheduler failed: %v", err)
	}

	scheduler.cron.Start()
	logx.Info("WafScheduler started")
}

func (scheduler *WafScheduler) Stop() {
	if scheduler == nil {
		return
	}

	scheduler.mu.Lock()
	if !scheduler.started {
		scheduler.mu.Unlock()
		return
	}
	scheduler.started = false
	scheduler.mu.Unlock()

	scheduler.cron.Stop()
	scheduler.removeAllEntries()
	logx.Info("WafScheduler stopped")
}

func (scheduler *WafScheduler) Reload() error {
	if scheduler == nil {
		return nil
	}
	if scheduler.db == nil {
		return fmt.Errorf("waf scheduler db is nil")
	}

	var sources []model.WafSource
	if err := scheduler.db.
		Where("enabled = ? AND COALESCE(schedule, '') <> ''", true).
		Order("id asc").
		Find(&sources).Error; err != nil {
		return fmt.Errorf("query scheduled waf sources failed: %w", err)
	}

	scheduler.removeAllEntries()
	for i := range sources {
		source := sources[i]
		if !shouldScheduleWafSource(&source) {
			continue
		}
		if err := scheduler.addOrUpdateSourceEntry(&source); err != nil {
			logx.Errorf("add scheduled waf source failed: id=%d name=%s err=%v", source.ID, source.Name, err)
		}
	}
	return nil
}

func (scheduler *WafScheduler) ReloadSource(sourceID uint) error {
	if scheduler == nil || sourceID == 0 {
		return nil
	}
	if scheduler.db == nil {
		return fmt.Errorf("waf scheduler db is nil")
	}

	var source model.WafSource
	if err := scheduler.db.First(&source, sourceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			scheduler.RemoveSource(sourceID)
			return nil
		}
		return fmt.Errorf("query waf source failed: %w", err)
	}

	if !shouldScheduleWafSource(&source) {
		scheduler.RemoveSource(sourceID)
		return nil
	}
	return scheduler.addOrUpdateSourceEntry(&source)
}

func (scheduler *WafScheduler) RemoveSource(sourceID uint) {
	if scheduler == nil || sourceID == 0 {
		return
	}
	if entryID, ok := scheduler.entryMap.Load(sourceID); ok {
		scheduler.cron.Remove(entryID.(cron.EntryID))
		scheduler.entryMap.Delete(sourceID)
	}
}

func (scheduler *WafScheduler) TriggerSourceNow(sourceID uint) error {
	if scheduler == nil || sourceID == 0 {
		return nil
	}
	if scheduler.db == nil {
		return fmt.Errorf("waf scheduler db is nil")
	}

	var source model.WafSource
	if err := scheduler.db.First(&source, sourceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("waf source not found")
		}
		return fmt.Errorf("query waf source failed: %w", err)
	}
	if !source.Enabled {
		return fmt.Errorf("waf source is disabled")
	}
	go scheduler.executeSource(sourceID)
	return nil
}

func (scheduler *WafScheduler) addOrUpdateSourceEntry(source *model.WafSource) error {
	if scheduler == nil || source == nil {
		return nil
	}

	scheduler.RemoveSource(source.ID)
	entryID, err := scheduler.cron.AddFunc(strings.TrimSpace(source.Schedule), func() {
		scheduler.executeSource(source.ID)
	})
	if err != nil {
		return err
	}
	scheduler.entryMap.Store(source.ID, entryID)
	return nil
}

func (scheduler *WafScheduler) executeSource(sourceID uint) {
	if scheduler == nil || sourceID == 0 {
		return
	}

	scheduler.mu.RLock()
	executor := scheduler.executor
	scheduler.mu.RUnlock()
	if executor == nil {
		logx.Errorf("skip scheduled waf source execution: executor is nil, sourceID=%d", sourceID)
		return
	}

	var source model.WafSource
	if err := scheduler.db.First(&source, sourceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			scheduler.RemoveSource(sourceID)
		}
		logx.Errorf("query scheduled waf source failed: sourceID=%d err=%v", sourceID, err)
		return
	}
	if !shouldScheduleWafSource(&source) {
		scheduler.RemoveSource(sourceID)
		return
	}

	execCtx := context.Background()
	if source.AutoCheck {
		if err := executor.CheckSource(execCtx, source.ID); err != nil {
			logx.Errorf("scheduled waf source check failed: sourceID=%d err=%v", source.ID, err)
			return
		}
	}

	if source.AutoDownload {
		if err := executor.SyncSource(execCtx, source.ID, false); err != nil {
			logx.Errorf("scheduled waf source sync failed: sourceID=%d err=%v", source.ID, err)
			return
		}
	}
}

func (scheduler *WafScheduler) removeAllEntries() {
	if scheduler == nil {
		return
	}
	scheduler.entryMap.Range(func(key, value any) bool {
		entryID, ok := value.(cron.EntryID)
		if ok {
			scheduler.cron.Remove(entryID)
		}
		scheduler.entryMap.Delete(key)
		return true
	})
}

func shouldScheduleWafSource(source *model.WafSource) bool {
	if source == nil {
		return false
	}
	if !source.Enabled {
		return false
	}
	if strings.TrimSpace(source.Schedule) == "" {
		return false
	}
	if normalizeWafSourceKind(source.Kind) == "coraza_engine" {
		return false
	}
	return source.AutoCheck || source.AutoDownload
}

func normalizeWafSourceKind(kind string) string {
	trimmed := strings.ToLower(strings.TrimSpace(kind))
	if trimmed == "" {
		return "crs"
	}
	return trimmed
}
