package tasks

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ArchiveTask 日志归档任务
type ArchiveTask struct {
	db           *gorm.DB
	retentionDay int
	enabled      bool
}

// NewArchiveTask 创建归档任务
func NewArchiveTask(db *gorm.DB, retentionDay int, enabled bool) *ArchiveTask {
	return &ArchiveTask{
		db:           db,
		retentionDay: retentionDay,
		enabled:      enabled,
	}
}

// Start 启动归档任务（每天凌晨 2 点执行）
func (t *ArchiveTask) Start(ctx context.Context) {
	if !t.enabled {
		fmt.Println("Archive task is disabled")
		return
	}

	fmt.Printf("Archive task started, retention days: %d\n", t.retentionDay)

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
			fmt.Println("Archive task stopped")
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
	fmt.Println("Starting log archiving...")
	startTime := time.Now()

	archiveDate := time.Now().AddDate(0, 0, -t.retentionDay)

	// 调用存储过程
	var archivedCount int
	err := t.db.Raw("SELECT archive_old_logs(?)", t.retentionDay).Scan(&archivedCount).Error

	if err != nil {
		fmt.Printf("Archive failed: %v\n", err)
		return
	}

	duration := time.Since(startTime)
	fmt.Printf("Archive completed: %d records moved to archive table (before %s), took %v\n",
		archivedCount, archiveDate.Format("2006-01-02"), duration)
}
