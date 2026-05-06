package tasks

import (
	"context"
	"logflux/internal/utils/safego"
	cronmodel "logflux/model/cron"
	"os/exec"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// Executor 负责执行调度器触发的任务。
type Executor interface {
	ExecuteTask(ctx context.Context, taskID uint, triggerMode string)
}

type CronScheduler struct {
	cron     *cron.Cron
	db       *gorm.DB
	entryMap sync.Map // map[uint]cron.EntryID
	executor Executor
}

func NewCronScheduler(db *gorm.DB) *CronScheduler {
	// Second-level precision if needed, but standard cron is minute-level.
	// Using standard parser.
	c := cron.New(cron.WithSeconds())
	return &CronScheduler{
		cron: c,
		db:   db,
	}
}

func (s *CronScheduler) SetExecutor(executor Executor) {
	s.executor = executor
}

func (s *CronScheduler) Start() {
	s.loadTasks()
	s.cron.Start()
	logx.Info("Cron 调度器已启动")
}

func (s *CronScheduler) Stop() {
	s.cron.Stop()
	logx.Info("Cron 调度器已停止")
}

func (s *CronScheduler) loadTasks() {
	var tasks []cronmodel.CronTask
	if err := s.db.Where("status = ?", 1).Find(&tasks).Error; err != nil {
		logx.Errorf("加载 Cron 任务失败: %v", err)
		return
	}

	for _, task := range tasks {
		s.AddTask(&task)
	}
}

func (s *CronScheduler) AddTask(task *cronmodel.CronTask) error {
	s.RemoveTask(task.ID) // Remove existing if any (for updates)

	if task.Status != 1 {
		return nil
	}

	entryID, err := s.cron.AddFunc(task.Schedule, func() {
		s.runTask(task.ID, "schedule")
	})
	if err != nil {
		logx.Errorf("添加 Cron 任务失败: name=%s err=%v", task.Name, err)
		return err
	}

	s.entryMap.Store(task.ID, entryID)
	logx.Infof("已添加 Cron 任务: %s (ID: %d, 调度: %s)", task.Name, task.ID, task.Schedule)
	return nil
}

func (s *CronScheduler) RemoveTask(taskID uint) {
	if val, ok := s.entryMap.Load(taskID); ok {
		s.cron.Remove(val.(cron.EntryID))
		s.entryMap.Delete(taskID)
		logx.Infof("已移除 Cron 任务: ID=%d", taskID)
	}
}

func (s *CronScheduler) TriggerTask(taskID uint) {
	safego.New(context.Background(), "手动触发定时任务").Go(func() {
		s.runTask(taskID, "manual")
	})
}

func (s *CronScheduler) runTask(taskID uint, triggerMode string) {
	if s != nil && s.executor != nil {
		s.executor.ExecuteTask(context.Background(), taskID, triggerMode)
		return
	}
	s.executeLegacyTask(taskID)
}

func (s *CronScheduler) executeLegacyTask(taskID uint) {
	var task cronmodel.CronTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		logx.Errorf("任务执行失败，未找到任务: %d", taskID)
		return
	}

	logEntry := cronmodel.CronTaskLog{
		TaskID:    task.ID,
		StartTime: time.Now(),
		Status:    0, // Running
	}
	s.db.Create(&logEntry)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(task.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", task.Script)
	// For Windows support (development environment), maybe fallback to PowerShell or errors?
	// Given user is on Windows, let's try purely shell but note it might fail without WSL/Git Bash.
	// Actually for cross-platform compatibility or simple testing on Windows,
	// we might check OS. But assuming production is Linux/Docker.
	// If local dev is Windows, "sh" might not exist.

	// Simple fix for Windows Dev: use "cmd" or "pwsh" if "sh" fails?
	// Or just assume user has git bash in path?
	// Let's stick to "sh" as per typical unix server reqs, but maybe wrap error.

	output, err := cmd.CombinedOutput()

	logEntry.EndTime = time.Now()
	logEntry.Duration = logEntry.EndTime.Sub(logEntry.StartTime).Milliseconds()
	logEntry.Output = string(output)

	if err != nil {
		logEntry.Status = 2 // Failed
		logEntry.Error = err.Error()
		if ctx.Err() == context.DeadlineExceeded {
			logEntry.Status = 3 // Timeout
			logEntry.Error = "Execution timed out"
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			logEntry.ExitCode = exitError.ExitCode()
		} else {
			logEntry.ExitCode = -1
		}
	} else {
		logEntry.Status = 1 // Success
		logEntry.ExitCode = 0
	}

	s.db.Save(&logEntry)
	logx.Infof("任务 %s 执行完成，状态: %d，耗时: %dms", task.Name, logEntry.Status, logEntry.Duration)
}
