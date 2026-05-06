package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	cronmodel "logflux/model/cron"
	"os"
	"path/filepath"
	"strings"
	"time"

	cronutil "logflux/common/cron"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CronService 负责定时任务业务、脚本文件历史与日志查询。
type CronService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCronService(ctx context.Context, svcCtx *svc.ServiceContext) *CronService {
	return &CronService{
		Logger: logger.New(logger.ModuleCron).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *CronService) CreateTask(req *types.CronTaskReq) (*types.BaseResp, error) {
	mode := cronutil.NormalizeScriptMode(req.ScriptMode)
	if err := cronutil.ValidateScriptMode(mode); err != nil {
		return nil, xerr.NewBusinessErrorWith("脚本来源类型无效")
	}
	if mode == cronutil.ScriptModeInline && strings.TrimSpace(req.Script) == "" {
		return nil, xerr.NewBusinessErrorWith("手写脚本不能为空")
	}

	task := &cronmodel.CronTask{
		Name:          strings.TrimSpace(req.Name),
		Schedule:      strings.TrimSpace(req.Schedule),
		ScriptMode:    mode,
		Script:        strings.TrimSpace(req.Script),
		CurrentFileID: 0,
		Status:        req.Status,
		Timeout:       req.Timeout,
	}
	if err := s.svcCtx.CronTaskModel.Create(s.ctx, task); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "创建定时任务失败", err)
	}
	if err := s.svcCtx.CronScheduler.AddTask(task); err != nil {
		s.Errorf("同步定时任务调度器失败: taskID=%d err=%v", task.ID, err)
	}
	return baseResp("创建成功"), nil
}

func (s *CronService) DeleteTask(req *types.IDReq) (*types.BaseResp, error) {
	task, err := s.svcCtx.CronTaskModel.FindByID(s.ctx, req.ID)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("定时任务不存在")
	}

	if err := s.deleteTaskData(task.ID); err != nil {
		return nil, err
	}

	s.svcCtx.CronScheduler.RemoveTask(task.ID)
	if err := os.RemoveAll(cronutil.TaskDir(cronutil.FilesBaseDir(&s.svcCtx.Config), task.ID)); err != nil {
		s.Errorf("清理定时任务脚本目录失败: taskID=%d err=%v", task.ID, err)
	}
	return baseResp("删除成功"), nil
}

func (s *CronService) GetLogList(req *types.CronLogListReq) (*types.CronLogListResp, error) {
	logs, total, err := s.svcCtx.CronTaskModel.ListLogs(s.ctx, req.TaskID, req.Status, req.Page, req.PageSize)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询定时任务日志失败", err)
	}

	list := make([]types.CronLogItem, 0, len(logs))
	for i := range logs {
		entry := logs[i]
		list = append(list, cronutil.CronLogItem(&entry))
	}
	return &types.CronLogListResp{List: list, Total: total}, nil
}

func (s *CronService) GetLogDetail(req *types.CronLogDetailReq) (*types.CronLogItem, error) {
	logEntry, err := s.svcCtx.CronTaskModel.FindLogByID(s.ctx, req.ID)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("执行日志不存在")
	}
	item := cronutil.CronLogItem(logEntry)
	return &item, nil
}

func (s *CronService) GetTaskList(req *types.CronTaskListReq) (*types.CronTaskListResp, error) {
	tasks, total, err := s.svcCtx.CronTaskModel.List(s.ctx, req.Name, req.Page, req.PageSize)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询定时任务列表失败", err)
	}

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	fileModel := cronmodel.NewCronTaskFileModel(s.svcCtx.DB)
	list := make([]types.CronTaskItem, 0, len(tasks))
	for i := range tasks {
		task := tasks[i]
		nextRun := ""
		if task.Status == 1 {
			if schedule, parseErr := parser.Parse(task.Schedule); parseErr == nil {
				nextRun = schedule.Next(time.Now()).Format("2006-01-02 15:04:05")
			}
		}

		var currentFile *cronmodel.CronTaskFile
		if task.CurrentFileID > 0 {
			if fileEntry, findErr := fileModel.FindByID(s.ctx, task.CurrentFileID); findErr == nil {
				currentFile = fileEntry
			}
		}
		list = append(list, cronutil.CronTaskItem(&task, currentFile, nextRun))
	}
	return &types.CronTaskListResp{List: list, Total: total}, nil
}

func (s *CronService) GetScriptHistory(req *types.CronTaskFileListReq) (*types.CronTaskFileListResp, error) {
	if _, err := s.svcCtx.CronTaskModel.FindByID(s.ctx, req.ID); err != nil {
		return nil, xerr.NewBusinessErrorWith("定时任务不存在")
	}

	files, total, err := s.svcCtx.CronTaskFileModel.ListByTaskID(s.ctx, req.ID, req.Page, req.PageSize)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询脚本历史失败", err)
	}

	list := make([]types.CronTaskFileItem, 0, len(files))
	for i := range files {
		fileEntry := files[i]
		list = append(list, cronutil.CronFileItem(&fileEntry))
	}
	return &types.CronTaskFileListResp{List: list, Total: total}, nil
}

func (s *CronService) TriggerTask(req *types.TriggerTaskReq) (*types.BaseResp, error) {
	task, err := s.svcCtx.CronTaskModel.FindByID(s.ctx, req.ID)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("定时任务不存在")
	}
	s.svcCtx.CronScheduler.TriggerTask(task.ID)
	return baseResp("触发成功"), nil
}

func (s *CronService) UpdateTask(req *types.CronTaskUpdateReq) (*types.BaseResp, error) {
	task, err := s.svcCtx.CronTaskModel.FindByID(s.ctx, req.ID)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("定时任务不存在")
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Schedule != "" {
		updates["schedule"] = strings.TrimSpace(req.Schedule)
	}
	if req.ScriptMode != "" {
		mode := cronutil.NormalizeScriptMode(req.ScriptMode)
		if err := cronutil.ValidateScriptMode(mode); err != nil {
			return nil, xerr.NewBusinessErrorWith("脚本来源类型无效")
		}
		if mode == cronutil.ScriptModeInline {
			effectiveScript := strings.TrimSpace(req.Script)
			if effectiveScript == "" {
				effectiveScript = strings.TrimSpace(task.Script)
			}
			if effectiveScript == "" {
				return nil, xerr.NewBusinessErrorWith("手写脚本不能为空")
			}
			if strings.TrimSpace(req.Script) != "" {
				updates["script"] = effectiveScript
			}
		} else if strings.TrimSpace(req.Script) != "" {
			updates["script"] = strings.TrimSpace(req.Script)
		}
		updates["script_mode"] = mode
		if mode == cronutil.ScriptModeInline {
			updates["current_file_id"] = 0
		}
	}
	if req.ScriptMode == "" && strings.TrimSpace(req.Script) != "" {
		updates["script"] = strings.TrimSpace(req.Script)
	}
	if req.Status >= 0 {
		updates["status"] = req.Status
	}
	if req.Timeout > 0 {
		updates["timeout"] = req.Timeout
	}
	if err := s.svcCtx.CronTaskModel.UpdateFields(s.ctx, task, updates); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新定时任务失败", err)
	}

	updatedTask, _ := s.svcCtx.CronTaskModel.FindByID(s.ctx, req.ID)
	if updatedTask != nil && updatedTask.Status == 1 {
		if err := s.svcCtx.CronScheduler.AddTask(updatedTask); err != nil {
			s.Errorf("同步定时任务调度器失败: taskID=%d err=%v", updatedTask.ID, err)
		}
	} else {
		s.svcCtx.CronScheduler.RemoveTask(req.ID)
	}
	return baseResp("更新成功"), nil
}

func (s *CronService) UploadTaskScript(req *types.CronTaskScriptUploadReq) (*types.BaseResp, error) {
	tempPath := cronutil.UploadTempPathFromContext(s.ctx)
	fileName := cronutil.UploadFileNameFromContext(s.ctx)
	if strings.TrimSpace(tempPath) == "" {
		return nil, xerr.NewBusinessErrorWith("上传文件不能为空")
	}
	if strings.TrimSpace(fileName) == "" {
		fileName = "script.sh"
	}

	baseDir := cronutil.FilesBaseDir(&s.svcCtx.Config)
	safeTempPath, err := s.ensureUploadPath(tempPath, baseDir)
	if err != nil {
		return nil, err
	}
	tempPath = safeTempPath
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if err := cronutil.EnsureWorkspace(baseDir); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "准备脚本目录失败", err)
	}

	data, readErr := os.ReadFile(tempPath)
	if readErr != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "读取上传脚本失败", readErr)
	}
	if int64(len(data)) <= 0 {
		return nil, xerr.NewBusinessErrorWith("上传文件不能为空")
	}
	if int64(len(data)) > 1024*1024 {
		return nil, xerr.NewBusinessErrorWith("上传文件过大")
	}

	safeName := cronutil.SafeFileName(fileName)
	var savedPath string
	var createdFile *cronmodel.CronTaskFile
	if err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		taskModel := cronmodel.NewCronTaskModel(tx)
		fileModel := cronmodel.NewCronTaskFileModel(tx)

		var currentTask cronmodel.CronTask
		if findErr := tx.WithContext(s.ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&currentTask, req.ID).Error; findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return xerr.NewBusinessErrorWith("定时任务不存在")
			}
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询定时任务失败", findErr)
		}

		latest, latestErr := fileModel.FindLatestByTaskID(s.ctx, currentTask.ID)
		nextVersion := 1
		if latestErr == nil && latest != nil {
			nextVersion = latest.Version + 1
		} else if latestErr != nil && !errors.Is(latestErr, gorm.ErrRecordNotFound) {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询脚本版本失败", latestErr)
		}

		taskDir := cronutil.TaskDir(baseDir, currentTask.ID)
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "准备脚本目录失败", err)
		}

		storedName := fmt.Sprintf("v%04d_%s", nextVersion, safeName)
		targetPath := filepath.Join(taskDir, storedName)
		savedPath = targetPath
		shaSum := sha256.Sum256(data)
		hexSHA := hex.EncodeToString(shaSum[:])
		if err := os.WriteFile(targetPath, data, 0o755); err != nil {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "保存脚本文件失败", err)
		}

		fileRecord := &cronmodel.CronTaskFile{
			TaskID:       currentTask.ID,
			Version:      nextVersion,
			OriginalName: fileName,
			StoredName:   storedName,
			FilePath:     targetPath,
			SizeBytes:    int64(len(data)),
			SHA256:       hexSHA,
			IsCurrent:    true,
		}
		if err := fileModel.Create(s.ctx, fileRecord); err != nil {
			_ = os.Remove(targetPath)
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "保存脚本版本失败", err)
		}
		createdFile = fileRecord

		if err := fileModel.ActivateByID(s.ctx, currentTask.ID, fileRecord.ID); err != nil {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "切换当前脚本版本失败", err)
		}

		updates := map[string]interface{}{
			"script_mode":     cronutil.ScriptModeFile,
			"current_file_id": fileRecord.ID,
		}
		if err := taskModel.UpdateFields(s.ctx, &currentTask, updates); err != nil {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新定时任务脚本信息失败", err)
		}

		return nil
	}); err != nil {
		if strings.TrimSpace(savedPath) != "" {
			_ = os.Remove(savedPath)
		}
		return nil, err
	}

	if createdFile != nil {
		s.Infof("脚本上传成功: taskID=%d fileID=%d version=%d file=%s", req.ID, createdFile.ID, createdFile.Version, createdFile.OriginalName)
	}
	return baseResp("上传成功"), nil
}

func (s *CronService) ActivateTaskScript(req *types.CronTaskFileActivateReq) (*types.BaseResp, error) {
	task, err := s.svcCtx.CronTaskModel.FindByID(s.ctx, req.ID)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("定时任务不存在")
	}

	fileEntry, err := s.svcCtx.CronTaskFileModel.FindByID(s.ctx, req.FileID)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("脚本版本不存在")
	}
	if fileEntry.TaskID != task.ID {
		return nil, xerr.NewBusinessErrorWith("脚本版本不属于当前任务")
	}

	if err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		taskModel := cronmodel.NewCronTaskModel(tx)
		fileModel := cronmodel.NewCronTaskFileModel(tx)

		currentTask, findErr := taskModel.FindByID(s.ctx, task.ID)
		if findErr != nil {
			return xerr.NewBusinessErrorWith("定时任务不存在")
		}
		if _, findErr = fileModel.FindByID(s.ctx, fileEntry.ID); findErr != nil {
			return xerr.NewBusinessErrorWith("脚本版本不存在")
		}
		if err := fileModel.ActivateByID(s.ctx, currentTask.ID, fileEntry.ID); err != nil {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "切换当前脚本版本失败", err)
		}
		updates := map[string]interface{}{
			"script_mode":     cronutil.ScriptModeFile,
			"current_file_id": fileEntry.ID,
		}
		if err := taskModel.UpdateFields(s.ctx, currentTask, updates); err != nil {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新定时任务脚本信息失败", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return baseResp("激活成功"), nil
}

func (s *CronService) deleteTaskData(taskID uint) error {
	return s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		taskModel := cronmodel.NewCronTaskModel(tx)
		fileModel := cronmodel.NewCronTaskFileModel(tx)

		if err := fileModel.DeleteByTaskID(s.ctx, taskID); err != nil {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "删除脚本历史失败", err)
		}
		if err := taskModel.DeleteByID(s.ctx, taskID); err != nil {
			return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "删除定时任务失败", err)
		}
		return nil
	})
}

func (s *CronService) ensureUploadPath(tempPath, baseDir string) (string, error) {
	tempPath = filepath.Clean(strings.TrimSpace(tempPath))
	if tempPath == "" {
		return "", xerr.NewBusinessErrorWith("上传文件不能为空")
	}
	if !cronutil.IsPathWithinBase(cronutil.TempDir(baseDir), tempPath) {
		return "", xerr.NewBusinessErrorWith("上传文件路径非法")
	}
	return tempPath, nil
}
