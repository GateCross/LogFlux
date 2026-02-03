package log

import (
	"context"
	"fmt"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyLogsLogic {
	return &GetCaddyLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyLogsLogic) GetCaddyLogs(req *types.CaddyLogReq) (resp *types.CaddyLogResp, err error) {
	var logs []model.CaddyLog
	var total int64

	db := l.svcCtx.DB.Model(&model.CaddyLog{})

	if req.Keyword != "" {
		like := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where(
			"host ILIKE ? OR uri ILIKE ? OR remote_ip ILIKE ? OR client_ip ILIKE ?",
			like, like, like, like,
		)
	}

	if req.Host != "" {
		db = db.Where("host = ?", strings.TrimSpace(req.Host))
	}

	if req.Status >= 0 {
		db = db.Where("status = ?", req.Status)
	}

	startTime, err := parseOptionalTime(req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid startTime: %w", err)
	}
	endTime, err := parseOptionalTime(req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid endTime: %w", err)
	}

	if startTime != nil {
		db = db.Where("log_time >= ?", *startTime)
	}
	if endTime != nil {
		db = db.Where("log_time <= ?", *endTime)
	}

	if err := db.Count(&total).Error; err != nil {
		l.Error("Count error: ", err)
		return nil, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("log_time desc, id desc").Limit(req.PageSize).Offset(offset).Find(&logs).Error; err != nil {
		l.Error("Find error: ", err)
		return nil, err
	}

	list := make([]types.CaddyLogItem, 0, len(logs))
	for _, logItem := range logs {
		list = append(list, types.CaddyLogItem{
			ID:        logItem.ID,
			LogTime:   logItem.LogTime.Format("2006-01-02 15:04:05"),
			Country:   logItem.Country,
			City:      logItem.City,
			Host:      logItem.Host,
			Method:    logItem.Method,
			Uri:       logItem.Uri,
			Status:    logItem.Status,
			Size:      logItem.Size,
			RemoteIP:  logItem.RemoteIP,
			ClientIP:  logItem.ClientIP,
			UserAgent: logItem.UserAgent,
		})
	}

	return &types.CaddyLogResp{
		List:  list,
		Total: total,
	}, nil
}

func parseOptionalTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("unsupported time format")
}
