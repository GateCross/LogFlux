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

	list := make([]types.CaddyLogItem, 0, len(logs))
	for _, logItem := range logs {
		location := strings.TrimSpace(strings.Join([]string{
			strings.TrimSpace(logItem.Country),
			strings.TrimSpace(logItem.Province),
			strings.TrimSpace(logItem.City),
		}, " "))
		list = append(list, types.CaddyLogItem{
			ID:        logItem.ID,
			LogTime:   logItem.LogTime.Format("2006-01-02 15:04:05"),
			Country:   logItem.Country,
			Province:  logItem.Province,
			City:      logItem.City,
			Location:  location,
			Host:      logItem.Host,
			Method:    logItem.Method,
			Uri:       logItem.Uri,
			Status:    logItem.Status,
			Size:      logItem.Size,
			RemoteIP:  logItem.RemoteIP,
			ClientIP:  logItem.ClientIP,
			UserAgent: logItem.UserAgent,
			RawLog:    logItem.RawLog,
		})
	}

	return &types.CaddyLogResp{
		List:  list,
		Total: total,
	}, nil
}
