package tasks

import (
	"context"
	"fmt"
	"time"

	"logflux/internal/notification"
	notificationmodel "logflux/model/notification"
	systemmodel "logflux/model/system"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 日志清理任务：每天凌晨 2 点按保留策略删除过期站点日志，并清理旧通知日志。
type LogCleanupTask struct {
	db              *gorm.DB
	notificationMgr notification.NotificationManager
}

// 单批删除行数，避免一次删除长时间占用表锁。
const logCleanupBatchSize = 2000

// 单轮最大批数上限，防止过期行数异常庞大时单轮长期占库；未删完的部分留待下一轮。
const logCleanupMaxBatches = 500

// 创建清理任务；任务无条件启动，是否清理由 system_configs 里的保留策略决定。
func NewLogCleanupTask(db *gorm.DB, nm notification.NotificationManager) *LogCleanupTask {
	return &LogCleanupTask{
		db:              db,
		notificationMgr: nm,
	}
}

// 启动清理任务，每天凌晨 2 点执行一次。
func (t *LogCleanupTask) Start(ctx context.Context) {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())

	timer := time.NewTimer(next.Sub(now))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			logx.Info("日志清理任务已停止")
			return
		case <-timer.C:
			t.runOnce()
			timer.Reset(24 * time.Hour)
		}
	}
}

// 执行一轮清理；读不到有效保留策略时跳过本轮，不做任何删除。
func (t *LogCleanupTask) runOnce() {
	logx.Info("开始日志清理...")
	startTime := time.Now()

	retention, err := systemmodel.LoadLogRetentionStrict(t.db)
	if err != nil {
		logx.Errorf("读取日志保留策略失败，跳过本轮清理: %v", err)
		return
	}
	if !retention.Enabled {
		logx.Info("日志保留策略已关闭，跳过本轮清理")
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retention.Days)
	deleted, err := t.deleteExpiredLogs(cutoff)
	if err != nil {
		logx.Errorf("日志清理失败: %v", err)
		if t.notificationMgr != nil {
			t.notificationMgr.Notify(context.Background(), notification.NewEvent(
				notification.EventArchiveFailed,
				notification.LevelError,
				"日志清理失败",
				fmt.Sprintf("日志清理任务执行出错: %v", err),
			))
		}
		return
	}

	// 清理过期的通知日志 (保留 30 天)
	retentionDate := time.Now().AddDate(0, 0, -30)
	if err := t.db.Where("created_at < ?", retentionDate).Delete(&notificationmodel.NotificationLog{}).Error; err != nil {
		logx.Errorf("清理旧通知日志失败: %v", err)
	} else {
		logx.Infof("已清理早于 %s 的通知日志", retentionDate.Format("2006-01-02"))
	}

	duration := time.Since(startTime)
	msg := fmt.Sprintf("日志清理完成: 已删除 %d 条早于 %s 的日志（保留 %d 天），耗时 %v",
		deleted, cutoff.Format("2006-01-02"), retention.Days, duration)
	logx.Info(msg)

	if t.notificationMgr != nil {
		t.notificationMgr.Notify(context.Background(), notification.NewEvent(
			notification.EventArchiveCompleted,
			notification.LevelInfo,
			"日志清理完成",
			msg,
		))
	}
}

// 分批删除早于 cutoff 的日志，返回删除总数。
// PostgreSQL 的 DELETE 不接受 LIMIT，用主键子查询限定每批行数。
func (t *LogCleanupTask) deleteExpiredLogs(cutoff time.Time) (int, error) {
	total := 0
	for batch := 1; ; batch++ {
		result := t.db.Exec(
			"DELETE FROM caddy_logs WHERE id IN (SELECT id FROM caddy_logs WHERE log_time < ? LIMIT ?)",
			cutoff, logCleanupBatchSize,
		)
		if result.Error != nil {
			return total, result.Error
		}
		if result.RowsAffected == 0 {
			return total, nil
		}
		total += int(result.RowsAffected)
		if batch >= logCleanupMaxBatches {
			logx.Errorf("日志清理单轮达到 %d 批上限，剩余过期日志留待下一轮", logCleanupMaxBatches)
			return total, nil
		}
	}
}
