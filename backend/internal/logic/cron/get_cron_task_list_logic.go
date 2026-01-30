package cron

import (
	"context"
	"logflux/common/result"
	"logflux/model"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronTaskListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCronTaskListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronTaskListLogic {
	return &GetCronTaskListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCronTaskListLogic) GetCronTaskList(req *types.CronTaskListReq) (resp *types.CronTaskListResp, err error) {
	var tasks []model.CronTask
	var total int64

	db := l.svcCtx.DB.Model(&model.CronTask{})
	if req.Name != "" {
		db = db.Where("name LIKE ?", "%"+req.Name+"%")
	}

	db.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := db.Offset(offset).Limit(req.PageSize).Order("id desc").Find(&tasks).Error; err != nil {
		return nil, result.NewCodeError(500, "Failed to get task list")
	}

	var list []types.CronTaskItem
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	for _, t := range tasks {
		nextRun := ""
		if t.Status == 1 {
			if schedule, err := parser.Parse(t.Schedule); err == nil {
				nextRun = schedule.Next(time.Now()).Format("2006-01-02 15:04:05")
			}
		}

		list = append(list, types.CronTaskItem{
			ID:        t.ID,
			Name:      t.Name,
			Schedule:  t.Schedule,
			Script:    t.Script,
			Status:    t.Status,
			Timeout:   t.Timeout,
			NextRun:   nextRun,
			CreatedAt: t.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: t.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.CronTaskListResp{
		List:  list,
		Total: total,
	}, nil
}
