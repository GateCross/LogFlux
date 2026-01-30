package tasks

import (
	"context"
	"logflux/model"
	"os/exec"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CronScheduler struct {
	cron     *cron.Cron
	db       *gorm.DB
	entryMap sync.Map // map[uint]cron.EntryID
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

func (s *CronScheduler) Start() {
	s.loadTasks()
	s.cron.Start()
	logx.Info("CronScheduler started")
}

func (s *CronScheduler) Stop() {
	s.cron.Stop()
	logx.Info("CronScheduler stopped")
}

func (s *CronScheduler) loadTasks() {
	var tasks []model.CronTask
	if err := s.db.Where("status = ?", 1).Find(&tasks).Error; err != nil {
		logx.Errorf("Failed to load cron tasks: %v", err)
		return
	}

	for _, task := range tasks {
		s.AddTask(&task)
	}
}

func (s *CronScheduler) AddTask(task *model.CronTask) error {
	s.RemoveTask(task.ID) // Remove existing if any (for updates)

	if task.Status != 1 {
		return nil
	}

	entryID, err := s.cron.AddFunc(task.Schedule, func() {
		s.executeTask(task.ID)
	})
	if err != nil {
		logx.Errorf("Failed to add cron task %s: %v", task.Name, err)
		return err
	}

	s.entryMap.Store(task.ID, entryID)
	logx.Infof("Added cron task: %s (ID: %d, Schedule: %s)", task.Name, task.ID, task.Schedule)
	return nil
}

func (s *CronScheduler) RemoveTask(taskID uint) {
	if val, ok := s.entryMap.Load(taskID); ok {
		s.cron.Remove(val.(cron.EntryID))
		s.entryMap.Delete(taskID)
		logx.Infof("Removed cron task ID: %d", taskID)
	}
}

func (s *CronScheduler) TriggerTask(taskID uint) {
	go s.executeTask(taskID)
}

func (s *CronScheduler) executeTask(taskID uint) {
	var task model.CronTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		logx.Errorf("Task execution failed, task not found: %d", taskID)
		return
	}

	logEntry := model.CronTaskLog{
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
	logx.Infof("Task %s executed. Status: %d, Duration: %dms", task.Name, logEntry.Status, logEntry.Duration)
}
