package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNotificationLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetNotificationLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNotificationLogsLogic {
	return &GetNotificationLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNotificationLogsLogic) GetNotificationLogs(req *types.LogListReq) (resp *types.LogListResp, err error) {
	var logs []model.NotificationLog
	db := l.svcCtx.DB.Model(&model.NotificationLog{})

	// 过滤条件
	if req.Status != 0 {
		switch req.Status {
		case 1:
			db = db.Where("status = ?", model.NotificationStatusPending)
		case 2:
			db = db.Where("status = ?", model.NotificationStatusSuccess)
		case 3:
			db = db.Where("status = ?", model.NotificationStatusFailed)
		}
	}

	if req.ChannelID != 0 {
		db = db.Where("channel_id = ?", req.ChannelID)
	}
	if req.RuleID != 0 {
		db = db.Where("rule_id = ?", req.RuleID)
	}

	// 分页
	var total int64
	db.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("id desc").Limit(req.PageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, err
	}

	list := make([]types.LogItem, 0, len(logs))
	for _, log := range logs {
		// 映射 status string 到 int
		statusInt := 0
		switch log.Status {
		case model.NotificationStatusPending:
			statusInt = 1
		case model.NotificationStatusSuccess:
			statusInt = 2
		case model.NotificationStatusFailed:
			statusInt = 3
		}

		item := types.LogItem{
			ID:         log.ID,
			EventID:    "", // Model doesn't have EventID
			EventType:  log.EventType,
			Title:      "",
			Message:    "",
			Level:      "",
			ChannelID:  log.ChannelID,
			Status:     statusInt,
			Error:      log.ErrorMessage,
			RetryCount: 0, // Model doesn't have RetryCount
			CreatedAt:  log.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if log.RuleID != nil {
			item.RuleID = uint(*log.RuleID)
		}
		if log.SentAt != nil {
			item.SentAt = log.SentAt.Format("2006-01-02 15:04:05")
		}

		// Safe assertions for map values
		if log.EventData != nil {
			if val, ok := log.EventData["title"]; ok {
				if strVal, ok := val.(string); ok {
					item.Title = strVal
				}
			}
			if val, ok := log.EventData["message"]; ok {
				if strVal, ok := val.(string); ok {
					item.Message = strVal
				}
			}
			if val, ok := log.EventData["level"]; ok {
				if strVal, ok := val.(string); ok {
					item.Level = strVal
				}
			}
		}

		list = append(list, item)
	}

	return &types.LogListResp{
		List:  list,
		Total: total,
	}, nil
}
