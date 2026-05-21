package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	cronmodel "logflux/model/cron"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cronutil "logflux/common/cron"
	"logflux/internal/utils/logger"

	"gorm.io/gorm"
)

// CronTaskExecutor 负责执行定时任务脚本并写入执行日志。
type CronTaskExecutor struct {
	db           *gorm.DB
	cronFilesDir string
}

// NewCronTaskExecutor 创建定时任务执行器。
func NewCronTaskExecutor(db *gorm.DB, cronFilesDir string) *CronTaskExecutor {
	cronFilesDir = strings.TrimSpace(cronFilesDir)
	if cronFilesDir == "" {
		cronFilesDir = cronutil.DefaultFilesDir
	}
	return &CronTaskExecutor{
		db:           db,
		cronFilesDir: filepath.Clean(cronFilesDir),
	}
}

// ExecuteTask 执行指定定时任务。
func (e *CronTaskExecutor) ExecuteTask(ctx context.Context, taskID uint, triggerMode string) {
	if e == nil || e.db == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	log := logger.New(logger.ModuleCron).WithContext(ctx)
	taskModel := cronmodel.NewCronTaskModel(e.db)
	fileModel := cronmodel.NewCronTaskFileModel(e.db)

	task, err := taskModel.FindByID(ctx, taskID)
	if err != nil {
		log.Errorf("执行定时任务失败，任务不存在: taskID=%d err=%v", taskID, err)
		return
	}

	mode := cronutil.ReadTaskScriptMode(task)
	logEntry := &cronmodel.CronTaskLog{
		TaskID:         task.ID,
		TaskName:       task.Name,
		TriggerMode:    cronutil.NormalizeTriggerMode(triggerMode),
		ScriptMode:     mode,
		StartTime:      timeNow(),
		Status:         0,
		ScriptSnapshot: cronutil.DefaultScriptText(task.Script),
	}

	if mode == cronutil.ScriptModeFile {
		currentFileID := task.CurrentFileID
		if currentFileID == 0 {
			if currentFile, findErr := fileModel.FindCurrentByTaskID(ctx, task.ID); findErr == nil {
				currentFileID = currentFile.ID
			}
		}
		if currentFileID > 0 {
			logEntry.ScriptFileID = currentFileID
			if fileEntry, findErr := fileModel.FindByID(ctx, currentFileID); findErr == nil {
				logEntry.ScriptFileVersion = fileEntry.Version
				logEntry.ScriptFileName = fileEntry.OriginalName
				logEntry.ScriptFilePath = fileEntry.FilePath
				logEntry.ScriptFileSHA256 = fileEntry.SHA256
				logEntry.ScriptSnapshot = ""
			}
		}
	}

	if err := taskModel.CreateLog(ctx, logEntry); err != nil {
		log.Errorf("创建定时任务执行日志失败: taskID=%d err=%v", taskID, err)
		return
	}

	execCtx, cancel := context.WithTimeout(ctx, cronutil.ExecutionTimeout(task.Timeout))
	defer cancel()

	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	runErr := e.runCommand(execCtx, task, logEntry, stdoutBuf, stderrBuf)

	logEntry.EndTime = timeNow()
	logEntry.Duration = logEntry.EndTime.Sub(logEntry.StartTime).Milliseconds()
	logEntry.Output = strings.TrimRight(stdoutBuf.String(), "\n")
	logEntry.Error = strings.TrimRight(stderrBuf.String(), "\n")

	if runErr != nil {
		logEntry.Status = 2
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			logEntry.Status = 3
			if logEntry.Error == "" {
				logEntry.Error = "执行超时"
			} else {
				logEntry.Error = logEntry.Error + "\n执行超时"
			}
		} else if logEntry.Error == "" {
			logEntry.Error = runErr.Error()
		}

		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			logEntry.ExitCode = exitErr.ExitCode()
		} else {
			logEntry.ExitCode = -1
		}
	} else {
		logEntry.Status = 1
		logEntry.ExitCode = 0
	}

	if err := taskModel.UpdateLog(ctx, logEntry); err != nil {
		log.Errorf("更新定时任务执行日志失败: logID=%d err=%v", logEntry.ID, err)
		return
	}

	log.Infof("定时任务执行完成: taskID=%d taskName=%s triggerMode=%s status=%d duration=%dms", task.ID, task.Name, logEntry.TriggerMode, logEntry.Status, logEntry.Duration)
}

func (e *CronTaskExecutor) runCommand(ctx context.Context, task *cronmodel.CronTask, logEntry *cronmodel.CronTaskLog, stdoutBuf, stderrBuf *bytes.Buffer) error {
	if task == nil {
		return fmt.Errorf("任务不存在")
	}

	var cmd *exec.Cmd
	switch cronutil.ReadTaskScriptMode(task) {
	case cronutil.ScriptModeFile:
		if logEntry == nil || strings.TrimSpace(logEntry.ScriptFilePath) == "" {
			return fmt.Errorf("脚本文件不存在")
		}
		if !cronutil.IsPathWithinBase(e.cronFilesDir, logEntry.ScriptFilePath) {
			return fmt.Errorf("脚本文件路径非法")
		}
		if _, err := os.Stat(logEntry.ScriptFilePath); err != nil {
			return fmt.Errorf("脚本文件不存在: %w", err)
		}
		ext := strings.ToLower(filepath.Ext(logEntry.ScriptFilePath))
		switch ext {
		case ".py":
			pythonCmd := "python3"
			if _, err := exec.LookPath("python3"); err != nil {
				if _, err2 := exec.LookPath("python"); err2 == nil {
					pythonCmd = "python"
				}
			}
			cmd = exec.CommandContext(ctx, pythonCmd, logEntry.ScriptFilePath)
		case ".go":
			cmd = exec.CommandContext(ctx, "go", "run", logEntry.ScriptFilePath)
		case ".bash":
			shellCmd := "bash"
			if _, err := exec.LookPath("bash"); err != nil {
				shellCmd = "sh"
			}
			cmd = exec.CommandContext(ctx, shellCmd, logEntry.ScriptFilePath)
		default:
			cmd = exec.CommandContext(ctx, "sh", logEntry.ScriptFilePath)
		}
		cmd.Dir = filepath.Dir(logEntry.ScriptFilePath)
	default:
		script := strings.TrimSpace(task.Script)
		if script == "" {
			return fmt.Errorf("执行脚本不能为空")
		}
		cmd = exec.CommandContext(ctx, "sh", "-c", script)
	}

	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	return cmd.Run()
}

func timeNow() time.Time {
	return time.Now()
}
