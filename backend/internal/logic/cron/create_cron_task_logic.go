package cron

import (
	"context"
	"logflux/common/result"
	"logflux/model"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateCronTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCronTaskLogic {
	return &CreateCronTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateCronTaskLogic) CreateCronTask(req *types.CronTaskReq) (resp *types.BaseResp, err error) {
	task := model.CronTask{
		Name:     req.Name,
		Schedule: req.Schedule,
		Script:   req.Script,
		Status:   req.Status,
		Timeout:  req.Timeout,
	}

	if err := l.svcCtx.DB.Create(&task).Error; err != nil {
		return nil, result.NewCodeError(500, "Failed to create task")
	}

	// Update Scheduler
	if err := l.svcCtx.CronScheduler.AddTask(&task); err != nil {
		l.Logger.Errorf("Failed to schedule task: %v", err)
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "Success",
	}, nil
}
