package cron

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"logflux/internal/config"
	"logflux/internal/types"
	cronmodel "logflux/model/cron"
)

const (
	ScriptModeInline = "inline"
	ScriptModeFile   = "file"

	TriggerModeManual   = "manual"
	TriggerModeSchedule = "schedule"

	DefaultFilesDir = "/config/cron-files"
)

type uploadContextKey string

const (
	uploadTempPathKey uploadContextKey = "cron_upload_temp_path"
	uploadFileNameKey uploadContextKey = "cron_upload_file_name"
)

func FilesBaseDir(c *config.Config) string {
	if c != nil {
		if dir := strings.TrimSpace(c.CronFilesDir); dir != "" {
			return filepath.Clean(dir)
		}
	}
	return DefaultFilesDir
}

func EnsureWorkspace(baseDir string) error {
	baseDir = filepath.Clean(strings.TrimSpace(baseDir))
	if baseDir == "" {
		return fmt.Errorf("定时任务脚本目录为空")
	}
	for _, dir := range []string{
		baseDir,
		filepath.Join(baseDir, "tasks"),
		filepath.Join(baseDir, "staging"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建目录失败: %s, %w", dir, err)
		}
	}
	return nil
}

func TaskDir(baseDir string, taskID uint) string {
	return filepath.Join(filepath.Clean(strings.TrimSpace(baseDir)), "tasks", fmt.Sprintf("%d", taskID))
}

func TempDir(baseDir string) string {
	return filepath.Join(filepath.Clean(strings.TrimSpace(baseDir)), "staging")
}

func SafeFileName(name string) string {
	base := strings.TrimSpace(name)
	base = filepath.Base(base)
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.ReplaceAll(base, "..", "_")
	base = strings.TrimSpace(base)
	if base == "" {
		return "script.sh"
	}
	return base
}

func NormalizeScriptMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", ScriptModeInline:
		return ScriptModeInline
	case ScriptModeFile:
		return ScriptModeFile
	default:
		return mode
	}
}

func ValidateScriptMode(mode string) error {
	switch NormalizeScriptMode(mode) {
	case ScriptModeInline, ScriptModeFile:
		return nil
	default:
		return fmt.Errorf("脚本来源类型无效")
	}
}

func ScriptModeLabel(mode string) string {
	switch NormalizeScriptMode(mode) {
	case ScriptModeFile:
		return "文件脚本"
	default:
		return "内联脚本"
	}
}

func DefaultScriptText(script string) string {
	return strings.TrimSpace(script)
}

func ReadTaskScriptMode(task *cronmodel.CronTask) string {
	if task == nil {
		return ScriptModeInline
	}
	return NormalizeScriptMode(task.ScriptMode)
}

func WithUploadTempPath(ctx context.Context, tempPath string) context.Context {
	return context.WithValue(ctx, uploadTempPathKey, tempPath)
}

func WithUploadFileName(ctx context.Context, fileName string) context.Context {
	return context.WithValue(ctx, uploadFileNameKey, fileName)
}

func UploadTempPathFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(uploadTempPathKey).(string)
	return strings.TrimSpace(value)
}

func UploadFileNameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(uploadFileNameKey).(string)
	return strings.TrimSpace(value)
}

func NormalizeTriggerMode(triggerMode string) string {
	triggerMode = strings.ToLower(strings.TrimSpace(triggerMode))
	switch triggerMode {
	case TriggerModeSchedule, TriggerModeManual:
		return triggerMode
	case "":
		return TriggerModeManual
	default:
		return triggerMode
	}
}

func ExecutionTimeout(timeout int) time.Duration {
	if timeout <= 0 {
		timeout = 60
	}
	return time.Duration(timeout) * time.Second
}

func IsPathWithinBase(baseDir, targetPath string) bool {
	baseDir = filepath.Clean(strings.TrimSpace(baseDir))
	targetPath = filepath.Clean(strings.TrimSpace(targetPath))
	if baseDir == "" || targetPath == "" {
		return false
	}

	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func CronFileItem(file *cronmodel.CronTaskFile) types.CronTaskFileItem {
	if file == nil {
		return types.CronTaskFileItem{}
	}
	return types.CronTaskFileItem{
		ID:           file.ID,
		TaskID:       file.TaskID,
		Version:      file.Version,
		OriginalName: file.OriginalName,
		StoredName:   file.StoredName,
		FilePath:     file.FilePath,
		SizeBytes:    file.SizeBytes,
		SHA256:       file.SHA256,
		IsCurrent:    file.IsCurrent,
		CreatedAt:    file.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func CronTaskItem(task *cronmodel.CronTask, currentFile *cronmodel.CronTaskFile, nextRun string) types.CronTaskItem {
	if task == nil {
		return types.CronTaskItem{}
	}
	item := types.CronTaskItem{
		ID:            task.ID,
		Name:          task.Name,
		Schedule:      task.Schedule,
		Script:        task.Script,
		ScriptMode:    NormalizeScriptMode(task.ScriptMode),
		CurrentFileID: task.CurrentFileID,
		Status:        task.Status,
		Timeout:       task.Timeout,
		NextRun:       nextRun,
		CreatedAt:     task.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     task.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if currentFile != nil {
		item.CurrentFileID = currentFile.ID
		item.CurrentFileName = currentFile.OriginalName
		item.CurrentFileVersion = currentFile.Version
		item.CurrentFilePath = currentFile.FilePath
		item.CurrentFileSHA256 = currentFile.SHA256
	}
	return item
}

func CronLogItem(logEntry *cronmodel.CronTaskLog) types.CronLogItem {
	if logEntry == nil {
		return types.CronLogItem{}
	}
	endTime := ""
	if !logEntry.EndTime.IsZero() {
		endTime = logEntry.EndTime.Format("2006-01-02 15:04:05")
	}
	return types.CronLogItem{
		ID:                logEntry.ID,
		TaskID:            logEntry.TaskID,
		TaskName:          logEntry.TaskName,
		StartTime:         logEntry.StartTime.Format("2006-01-02 15:04:05"),
		EndTime:           endTime,
		Status:            logEntry.Status,
		ExitCode:          logEntry.ExitCode,
		Output:            logEntry.Output,
		Error:             logEntry.Error,
		Duration:          logEntry.Duration,
		TriggerMode:       logEntry.TriggerMode,
		ScriptMode:        NormalizeScriptMode(logEntry.ScriptMode),
		ScriptSnapshot:    logEntry.ScriptSnapshot,
		ScriptFileID:      logEntry.ScriptFileID,
		ScriptFileVersion: logEntry.ScriptFileVersion,
		ScriptFileName:    logEntry.ScriptFileName,
		ScriptFilePath:    logEntry.ScriptFilePath,
		ScriptFileSHA256:  logEntry.ScriptFileSHA256,
	}
}
