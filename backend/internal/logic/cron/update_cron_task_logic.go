package cron

import (
	"context"
	"logflux/common/result"
	"logflux/model"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCronTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCronTaskLogic {
	return &UpdateCronTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCronTaskLogic) UpdateCronTask(req *types.CronTaskUpdateReq) (resp *types.BaseResp, err error) {
	var task model.CronTask
	if err := l.svcCtx.DB.First(&task, req.ID).Error; err != nil {
		return nil, result.NewCodeError(404, "Task not found")
	}

	// Dynamic update fields
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
	updates["status"] = req.Status // Status 0 is valid (disabled)
	if req.Timeout > 0 {
		updates["timeout"] = req.Timeout
	}

	if err := l.svcCtx.DB.Model(&task).Updates(updates).Error; err != nil {
		return nil, result.NewCodeError(500, "Failed to update task")
	}

	// Reload updated task to get full struct
	l.svcCtx.DB.First(&task, req.ID)

	// Update Scheduler
	if task.Status == 1 {
		l.svcCtx.CronScheduler.AddTask(&task)
	} else {
		l.svcCtx.CronScheduler.RemoveTask(task.ID)
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "Success",
	}, nil
}
