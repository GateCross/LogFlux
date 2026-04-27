package service

import (
	"context"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"
	"logflux/model"

	"github.com/robfig/cron/v3"
)

// CronService 负责定时任务业务和调度器同步。
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
	task := &model.CronTask{
		Name:     req.Name,
		Schedule: req.Schedule,
		Script:   req.Script,
		Status:   req.Status,
		Timeout:  req.Timeout,
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
	s.svcCtx.CronScheduler.RemoveTask(req.ID)
	if err := s.svcCtx.CronTaskModel.DeleteByID(s.ctx, req.ID); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "删除定时任务失败", err)
	}
	return baseResp("删除成功"), nil
}

func (s *CronService) GetLogList(req *types.CronLogListReq) (*types.CronLogListResp, error) {
	logs, total, err := s.svcCtx.CronTaskModel.ListLogs(s.ctx, req.TaskID, req.Status, req.Page, req.PageSize)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询定时任务日志失败", err)
	}

	list := make([]types.CronLogItem, 0, len(logs))
	for _, taskLog := range logs {
		endTime := ""
		if !taskLog.EndTime.IsZero() {
			endTime = taskLog.EndTime.Format("2006-01-02 15:04:05")
		}
		list = append(list, types.CronLogItem{
			ID:        taskLog.ID,
			TaskID:    taskLog.TaskID,
			TaskName:  taskLog.Task.Name,
			StartTime: taskLog.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:   endTime,
			Status:    taskLog.Status,
			ExitCode:  taskLog.ExitCode,
			Output:    taskLog.Output,
			Error:     taskLog.Error,
			Duration:  taskLog.Duration,
		})
	}
	return &types.CronLogListResp{List: list, Total: total}, nil
}

func (s *CronService) GetTaskList(req *types.CronTaskListReq) (*types.CronTaskListResp, error) {
	tasks, total, err := s.svcCtx.CronTaskModel.List(s.ctx, req.Name, req.Page, req.PageSize)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询定时任务列表失败", err)
	}

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	list := make([]types.CronTaskItem, 0, len(tasks))
	for _, task := range tasks {
		nextRun := ""
		if task.Status == 1 {
			if schedule, err := parser.Parse(task.Schedule); err == nil {
				nextRun = schedule.Next(time.Now()).Format("2006-01-02 15:04:05")
			}
		}
		list = append(list, types.CronTaskItem{
			ID:        task.ID,
			Name:      task.Name,
			Schedule:  task.Schedule,
			Script:    task.Script,
			Status:    task.Status,
			Timeout:   task.Timeout,
			NextRun:   nextRun,
			CreatedAt: task.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: task.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &types.CronTaskListResp{List: list, Total: total}, nil
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
		updates["name"] = req.Name
	}
	if req.Schedule != "" {
		updates["schedule"] = req.Schedule
	}
	if req.Script != "" {
		updates["script"] = req.Script
	}
	updates["status"] = req.Status
	if req.Timeout > 0 {
		updates["timeout"] = req.Timeout
	}
	if err := s.svcCtx.CronTaskModel.UpdateFields(s.ctx, task, updates); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新定时任务失败", err)
	}

	task, _ = s.svcCtx.CronTaskModel.FindByID(s.ctx, req.ID)
	if task != nil && task.Status == 1 {
		_ = s.svcCtx.CronScheduler.AddTask(task)
	} else {
		s.svcCtx.CronScheduler.RemoveTask(req.ID)
	}
	return baseResp("更新成功"), nil
}
