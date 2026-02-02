package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchDeleteNotificationLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchDeleteNotificationLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDeleteNotificationLogsLogic {
	return &BatchDeleteNotificationLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type BatchDeleteReq struct {
	IDs []uint `json:"ids"`
}

func (l *BatchDeleteNotificationLogsLogic) BatchDeleteNotificationLogs(req *BatchDeleteReq) (resp *types.BaseResp, err error) {
	if len(req.IDs) == 0 {
		return &types.BaseResp{Code: 200, Msg: "success"}, nil
	}

	l.svcCtx.DB.Where("log_id IN ?", req.IDs).Delete(&model.NotificationJob{})
	if err := l.svcCtx.DB.Where("id IN ?", req.IDs).Delete(&model.NotificationLog{}).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
