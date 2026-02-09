package tasks

import (
	"context"
	"fmt"
	"logflux/internal/notification"
	"logflux/model"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// ArchiveTask 日志归档任务
type ArchiveTask struct {
	db              *gorm.DB
	retentionDay    int
	enabled         bool
	notificationMgr notification.NotificationManager
}

// NewArchiveTask 创建归档任务
func NewArchiveTask(db *gorm.DB, retentionDay int, enabled bool, nm notification.NotificationManager) *ArchiveTask {
	return &ArchiveTask{
		db:              db,
		retentionDay:    retentionDay,
		enabled:         enabled,
		notificationMgr: nm,
	}
}

// Start 启动归档任务（每天凌晨 2 点执行）
func (t *ArchiveTask) Start(ctx context.Context) {
	if !t.enabled {
		logx.Info("Archive task is disabled")
		return
	}

	logx.Infof("Archive task started, retention days: %d", t.retentionDay)

	// 立即执行一次归档（可选）
	// t.runArchive()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// 计算到下一个凌晨 2 点的时间
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
	duration := next.Sub(now)

	// 等待到凌晨 2 点
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			logx.Info("Archive task stopped")
			return
		case <-timer.C:
			t.runArchive()
			timer.Reset(24 * time.Hour)
		case <-ticker.C:
			// 作为备份，每 24 小时也执行一次
			t.runArchive()
		}
	}
}

// runArchive 执行归档逻辑
func (t *ArchiveTask) runArchive() {
	logx.Info("Starting log archiving...")
	startTime := time.Now()

	archiveDate := time.Now().AddDate(0, 0, -t.retentionDay)

	// 调用存储过程
	var archivedCount int
	err := t.db.Raw("SELECT archive_old_logs(?)", t.retentionDay).Scan(&archivedCount).Error

	if err != nil {
		logx.Errorf("Archive failed: %v", err)
		// 发送失败通知
		if t.notificationMgr != nil {
			t.notificationMgr.Notify(context.Background(), notification.NewEvent(
				"system.archive.failed",
				notification.LevelError,
				"日志归档失败",
				fmt.Sprintf("归档任务执行出错: %v", err),
			))
		}
		return
	}

	// 清理过期的通知日志 (保留 30 天)
	retentionDate := time.Now().AddDate(0, 0, -30)
	if err := t.db.Where("created_at < ?", retentionDate).Delete(&model.NotificationLog{}).Error; err != nil {
		logx.Errorf("Failed to clean up old notification logs: %v", err)
	} else {
		logx.Infof("Cleaned up notification logs older than %s", retentionDate.Format("2006-01-02"))
	}

	duration := time.Since(startTime)
	msg := fmt.Sprintf("Archive completed: %d records moved to archive table (before %s), took %v",
		archivedCount, archiveDate.Format("2006-01-02"), duration)
	logx.Info(msg)

	// 发送成功通知 (仅当有数据归档或作为定期报告时)
	if t.notificationMgr != nil {
		t.notificationMgr.Notify(context.Background(), notification.NewEvent(
			"system.archive.success",
			notification.LevelInfo,
			"日志归档完成",
			msg,
		))
	}
}
