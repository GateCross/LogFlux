package log

import (
	"context"

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

	// Filtering
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		db = db.Where("host LIKE ? OR uri LIKE ? OR client_ip LIKE ? OR remote_ip LIKE ?",
			keyword, keyword, keyword, keyword)
	}

	// Count Total
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("log_time DESC").Offset(offset).Limit(req.PageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	// Response Mapping
	list := make([]types.CaddyLogItem, 0, len(logs))
	for _, log := range logs {
		list = append(list, types.CaddyLogItem{
			ID:        log.ID,
			LogTime:   log.LogTime.Format("2006-01-02 15:04:05.000"),
			Country:   log.Country,
			City:      log.City,
			Host:      log.Host,
			Method:    log.Method,
			Uri:       log.Uri,
			Status:    log.Status,
			Size:      log.Size,
			RemoteIP:  log.RemoteIP,
			ClientIP:  log.ClientIP,
			UserAgent: log.UserAgent,
		})
	}

	return &types.CaddyLogResp{
		List:  list,
		Total: total,
	}, nil
}
