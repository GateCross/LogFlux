package cron

import (
	"context"
	"logflux/common/result"
	"logflux/model"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TriggerCronTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTriggerCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TriggerCronTaskLogic {
	return &TriggerCronTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TriggerCronTaskLogic) TriggerCronTask(req *types.TriggerTaskReq) (resp *types.BaseResp, err error) {
	var task model.CronTask
	if err := l.svcCtx.DB.First(&task, req.ID).Error; err != nil {
		return nil, result.NewCodeError(404, "Task not found")
	}

	l.svcCtx.CronScheduler.TriggerTask(task.ID)

	return &types.BaseResp{
		Code: 200,
		Msg:  "Task triggered successfully",
	}, nil
}
