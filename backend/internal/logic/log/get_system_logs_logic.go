package log

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSystemLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSystemLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSystemLogsLogic {
	return &GetSystemLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSystemLogsLogic) GetSystemLogs(req *types.SystemLogReq) (resp *types.SystemLogResp, err error) {
	var logs []model.SystemLog
	var total int64

	db := l.svcCtx.DB.Model(&model.SystemLog{})

	if req.Keyword != "" {
		like := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where(
			"message ILIKE ? OR caller ILIKE ? OR raw_log ILIKE ?",
			like, like, like,
		)
	}

	if strings.TrimSpace(req.Source) != "" {
		db = db.Where("source = ?", strings.TrimSpace(req.Source))
	}

	if strings.TrimSpace(req.Level) != "" {
		db = db.Where("level = ?", strings.TrimSpace(strings.ToLower(req.Level)))
	}

	startTime, err := utils.ParseOptionalTime(req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid startTime: %w", err)
	}
	endTime, err := utils.ParseOptionalTime(req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid endTime: %w", err)
	}

	if startTime != nil {
		db = db.Where("log_time >= ?", *startTime)
	}
	if endTime != nil {
		db = db.Where("log_time <= ?", *endTime)
	}

	orderBy := "log_time desc, id desc"
	switch strings.ToLower(strings.TrimSpace(req.SortBy)) {
	case "logtime", "log_time", "time":
		if strings.ToLower(strings.TrimSpace(req.Order)) == "asc" {
			orderBy = "log_time asc, id asc"
		}
	}

	if err := db.Count(&total).Error; err != nil {
		l.Error("Count error: ", err)
		return nil, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := db.Order(orderBy).Limit(req.PageSize).Offset(offset).Find(&logs).Error; err != nil {
		l.Error("Find error: ", err)
		return nil, err
	}

	list := make([]types.SystemLogItem, 0, len(logs))
	for _, logItem := range logs {
		list = append(list, types.SystemLogItem{
			ID:        logItem.ID,
			LogTime:   logItem.LogTime.Format("2006-01-02 15:04:05"),
			Level:     logItem.Level,
			Message:   logItem.Message,
			Caller:    logItem.Caller,
			TraceID:   logItem.TraceID,
			SpanID:    logItem.SpanID,
			Source:    logItem.Source,
			RawLog:    logItem.RawLog,
			ExtraData: logItem.ExtraData,
		})
	}

	return &types.SystemLogResp{
		List:  list,
		Total: total,
	}, nil
}
