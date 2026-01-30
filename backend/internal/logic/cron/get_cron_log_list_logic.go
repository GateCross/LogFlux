package cron

import (
	"context"
	"logflux/common/result"
	"logflux/model"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronLogListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCronLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronLogListLogic {
	return &GetCronLogListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCronLogListLogic) GetCronLogList(req *types.CronLogListReq) (resp *types.CronLogListResp, err error) {
	var logs []model.CronTaskLog
	var total int64

	db := l.svcCtx.DB.Model(&model.CronTaskLog{})

	if req.TaskID > 0 {
		db = db.Where("task_id = ?", req.TaskID)
	}
	if req.Status >= 0 { // -1 for all
		db = db.Where("status = ?", req.Status)
	}

	db.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	// Preload Task to get Task Name
	if err := db.Preload("Task").Offset(offset).Limit(req.PageSize).Order("id desc").Find(&logs).Error; err != nil {
		return nil, result.NewCodeError(500, "Failed to get log list")
	}

	var list []types.CronLogItem
	for _, log := range logs {
		endTimeStr := ""
		if !log.EndTime.IsZero() {
			endTimeStr = log.EndTime.Format("2006-01-02 15:04:05")
		}

		list = append(list, types.CronLogItem{
			ID:        log.ID,
			TaskID:    log.TaskID,
			TaskName:  log.Task.Name,
			StartTime: log.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:   endTimeStr,
			Status:    log.Status,
			ExitCode:  log.ExitCode,
			Output:    log.Output,
			Error:     log.Error,
			Duration:  log.Duration,
		})
	}

	return &types.CronLogListResp{
		List:  list,
		Total: total,
	}, nil
}
