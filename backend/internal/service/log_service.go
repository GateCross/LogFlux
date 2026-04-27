package service

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"
	"logflux/model"
)

// LogService 负责日志查询与响应组装。
type LogService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogService(ctx context.Context, svcCtx *svc.ServiceContext) *LogService {
	return &LogService{
		Logger: logger.New(logger.ModuleLog).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *LogService) GetCaddyLogs(req *types.CaddyLogReq) (*types.CaddyLogResp, error) {
	startTime, err := utils.ParseOptionalTime(req.StartTime)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("开始时间格式无效: %v", err))
	}
	endTime, err := utils.ParseOptionalTime(req.EndTime)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("结束时间格式无效: %v", err))
	}

	logs, total, err := s.caddyLogModel().List(s.ctx, model.CaddyLogQuery{
		Keyword:  req.Keyword,
		Host:     req.Host,
		Status:   req.Status,
		Start:    startTime,
		End:      endTime,
		SortBy:   req.SortBy,
		Order:    req.Order,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询 Caddy 日志失败", err)
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
	return &types.CaddyLogResp{List: list, Total: total}, nil
}

func (s *LogService) GetSystemLogs(req *types.SystemLogReq) (*types.SystemLogResp, error) {
	startTime, err := utils.ParseOptionalTime(req.StartTime)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("开始时间格式无效: %v", err))
	}
	endTime, err := utils.ParseOptionalTime(req.EndTime)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("结束时间格式无效: %v", err))
	}

	logs, total, err := s.systemLogModel().List(s.ctx, model.SystemLogQuery{
		Keyword:  req.Keyword,
		Source:   req.Source,
		Level:    req.Level,
		Start:    startTime,
		End:      endTime,
		SortBy:   req.SortBy,
		Order:    req.Order,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询系统日志失败", err)
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
	return &types.SystemLogResp{List: list, Total: total}, nil
}

func (s *LogService) caddyLogModel() model.CaddyLogModel {
	if s.svcCtx.CaddyLogModel != nil {
		return s.svcCtx.CaddyLogModel
	}
	return model.NewCaddyLogModel(s.svcCtx.DB)
}

func (s *LogService) systemLogModel() model.SystemLogModel {
	if s.svcCtx.SystemLogModel != nil {
		return s.svcCtx.SystemLogModel
	}
	return model.NewSystemLogModel(s.svcCtx.DB)
}
