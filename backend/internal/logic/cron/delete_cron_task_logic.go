package cron

import (
	"context"
	"logflux/common/result"
	"logflux/model"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCronTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCronTaskLogic {
	return &DeleteCronTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteCronTaskLogic) DeleteCronTask(req *types.IDReq) (resp *types.BaseResp, err error) {
	// Remove from Scheduler first
	l.svcCtx.CronScheduler.RemoveTask(req.ID)

	// Delete from DB
	if err := l.svcCtx.DB.Delete(&model.CronTask{}, req.ID).Error; err != nil {
		return nil, result.NewCodeError(500, "Failed to delete task")
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "Success",
	}, nil
}
