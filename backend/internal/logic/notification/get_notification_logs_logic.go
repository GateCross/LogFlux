package notification

import (
	"context"
	"time"

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
	type logWithJob struct {
		model.NotificationLog
		JobStatus     string     `gorm:"column:job_status"`
		JobRetryCount int        `gorm:"column:job_retry_count"`
		JobNextRunAt  *time.Time `gorm:"column:job_next_run_at"`
		JobLastError  string     `gorm:"column:job_last_error"`
	}

	var logs []logWithJob
	db := l.svcCtx.DB.Model(&model.NotificationLog{}).
		Select("notification_logs.*, notification_jobs.status as job_status, notification_jobs.retry_count as job_retry_count, notification_jobs.next_run_at as job_next_run_at, notification_jobs.last_error as job_last_error").
		Joins("LEFT JOIN notification_jobs ON notification_jobs.log_id = notification_logs.id")

	// 过滤条件
	if req.Status >= 0 {
		switch req.Status {
		case 0:
			// 前端 0 = pending (排队/未发送)
			db = db.Where("status = ?", model.NotificationStatusPending)
		case 1:
			// 前端 1 = sending
			db = db.Where("status = ?", model.NotificationStatusSending)
		case 2:
			db = db.Where("status = ?", model.NotificationStatusSuccess)
		case 3:
			db = db.Where("status = ?", model.NotificationStatusFailed)
		}
	}

	if req.ChannelID != 0 {
		db = db.Where("notification_logs.channel_id = ?", req.ChannelID)
	}
	if req.RuleID != 0 {
		db = db.Where("notification_logs.rule_id = ?", req.RuleID)
	}
	if req.JobStatus != "" {
		db = db.Where("notification_jobs.status = ?", req.JobStatus)
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
		// 映射 status string 到 int（与前端一致）
		statusInt := 0
		switch log.Status {
		case model.NotificationStatusPending:
			statusInt = 0
		case model.NotificationStatusSending:
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
			ChannelID:  0,
			Status:     statusInt,
			Error:      log.ErrorMessage,
			RetryCount: 0,
			CreatedAt:  log.CreatedAt.Format("2006-01-02 15:04:05"),
			JobStatus:     log.JobStatus,
			JobRetryCount: log.JobRetryCount,
			LastError:     log.JobLastError,
		}
		if log.RuleID != nil {
			item.RuleID = uint(*log.RuleID)
		}
		if log.ChannelID != nil {
			item.ChannelID = uint(*log.ChannelID)
		}
		if log.SentAt != nil {
			item.SentAt = log.SentAt.Format("2006-01-02 15:04:05")
		}
		if log.JobNextRunAt != nil {
			item.NextRunAt = log.JobNextRunAt.Format("2006-01-02 15:04:05")
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
