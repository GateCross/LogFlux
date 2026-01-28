package log

import (
	"context"
	"encoding/json"
	"fmt"
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
	// 尝试从 Redis 缓存获取（如果启用）
	if l.svcCtx.Redis != nil {
		cacheKey := fmt.Sprintf("caddy_logs:page:%d:size:%d:keyword:%s", req.Page, req.PageSize, req.Keyword)
		cached, err := l.svcCtx.Redis.Get(l.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var cachedResp types.CaddyLogResp
			if json.Unmarshal([]byte(cached), &cachedResp) == nil {
				l.Logger.Info("Cache hit for caddy logs")
				return &cachedResp, nil
			}
		}
	}

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

	// Pagination - 使用优化的查询
	offset := (req.Page - 1) * req.PageSize
	// 使用索引优化的排序字段
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

	result := &types.CaddyLogResp{
		List:  list,
		Total: total,
	}

	// 存入 Redis 缓存（如果启用），缓存 5 分钟
	if l.svcCtx.Redis != nil {
		cacheKey := fmt.Sprintf("caddy_logs:page:%d:size:%d:keyword:%s", req.Page, req.PageSize, req.Keyword)
		if data, err := json.Marshal(result); err == nil {
			l.svcCtx.Redis.Set(l.ctx, cacheKey, string(data), 5*time.Minute)
		}
	}

	return result, nil
}
