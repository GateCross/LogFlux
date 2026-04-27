package service

import (
	"context"
	"fmt"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"
	"logflux/model"
)

// DashboardService 负责首页看板聚合查询。
type DashboardService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDashboardService(ctx context.Context, svcCtx *svc.ServiceContext) *DashboardService {
	return &DashboardService{
		Logger: logger.New(logger.ModuleSystem).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *DashboardService) GetSummary(req *types.DashboardSummaryReq) (*types.DashboardSummaryResp, error) {
	startTime, endTime, intervalSec, err := normalizeDashboardRange(req)
	if err != nil {
		return nil, err
	}
	topN := req.TopN
	if topN <= 0 {
		topN = 6
	}
	recentLimit := req.RecentLimit
	if recentLimit <= 0 {
		recentLimit = 6
	}

	blockedStatuses := []int{403, 429}
	logModel := s.caddyLogModel()
	total, err := logModel.CountRange(s.ctx, startTime, endTime)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计请求量失败", err)
	}
	blocked, err := logModel.CountStatuses(s.ctx, startTime, endTime, blockedStatuses)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计拦截请求失败", err)
	}
	err4xx, err := logModel.CountStatusRange(s.ctx, startTime, endTime, 400, 500)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计 4xx 请求失败", err)
	}
	err5xx, err := logModel.CountStatusRange(s.ctx, startTime, endTime, 500, 600)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计 5xx 请求失败", err)
	}
	uv, err := logModel.CountUniqueVisitor(s.ctx, startTime, endTime)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计访客数失败", err)
	}
	uniqueIP, err := logModel.CountUniqueRemoteIP(s.ctx, startTime, endTime)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计独立 IP 失败", err)
	}
	attackIP, err := logModel.CountAttackIP(s.ctx, startTime, endTime, blockedStatuses)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计攻击 IP 失败", err)
	}

	trend, err := s.loadTrendSeries(startTime, endTime, intervalSec)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询趋势数据失败", err)
	}
	geo, err := s.loadGeoStats(startTime, endTime, topN)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询地域数据失败", err)
	}
	recent, err := s.loadRecentLogs(startTime, endTime, recentLimit)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询最近日志失败", err)
	}

	return &types.DashboardSummaryResp{
		Stats: types.DashboardStats{
			Requests: total,
			PV:       total,
			UV:       uv,
			UniqueIP: uniqueIP,
			Blocked:  blocked,
			AttackIP: attackIP,
		},
		ErrorStats: types.DashboardErrorStats{
			Error4xx:   err4xx,
			Blocked4xx: blocked,
			Error5xx:   err5xx,
		},
		Trend:  trend,
		Geo:    geo,
		Recent: recent,
		Range: types.DashboardRange{
			StartTime:   startTime.Format("2006-01-02 15:04:05"),
			EndTime:     endTime.Format("2006-01-02 15:04:05"),
			IntervalSec: intervalSec,
		},
	}, nil
}

func normalizeDashboardRange(req *types.DashboardSummaryReq) (time.Time, time.Time, int, error) {
	start, err := utils.ParseOptionalTime(req.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, 0, xerr.NewBusinessErrorWith(fmt.Sprintf("开始时间格式无效: %v", err))
	}
	end, err := utils.ParseOptionalTime(req.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, 0, xerr.NewBusinessErrorWith(fmt.Sprintf("结束时间格式无效: %v", err))
	}

	now := time.Now()
	if end == nil {
		end = &now
	}
	if start == nil {
		oneHour := end.Add(-1 * time.Hour)
		start = &oneHour
	}
	if start.After(*end) {
		start, end = end, start
	}

	intervalSec := req.IntervalSec
	if intervalSec <= 0 {
		intervalSec = 60
	}
	return *start, *end, intervalSec, nil
}

func (s *DashboardService) loadTrendSeries(startTime, endTime time.Time, intervalSec int) ([]types.DashboardTrendItem, error) {
	rows, err := s.caddyLogModel().TrendRows(s.ctx, startTime, endTime, intervalSec)
	if err != nil {
		return nil, err
	}

	bucketMap := make(map[int64]int64, len(rows))
	for _, row := range rows {
		bucketMap[row.Bucket] = row.Count
	}

	bucketStart := (startTime.Unix() / int64(intervalSec)) * int64(intervalSec)
	bucketEnd := (endTime.Unix() / int64(intervalSec)) * int64(intervalSec)
	labelLayout := "15:04"
	if endTime.Sub(startTime) > 24*time.Hour {
		labelLayout = "01-02 15:04"
	}

	series := make([]types.DashboardTrendItem, 0)
	for ts := bucketStart; ts <= bucketEnd; ts += int64(intervalSec) {
		series = append(series, types.DashboardTrendItem{
			Time:  time.Unix(ts, 0).Format(labelLayout),
			Value: bucketMap[ts],
		})
	}
	return series, nil
}

func (s *DashboardService) loadGeoStats(startTime, endTime time.Time, topN int) ([]types.DashboardGeoItem, error) {
	rows, err := s.caddyLogModel().GeoRows(s.ctx, startTime, endTime, topN)
	if err != nil {
		return nil, err
	}
	items := make([]types.DashboardGeoItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.DashboardGeoItem{Name: row.Name, Value: row.Value})
	}
	return items, nil
}

func (s *DashboardService) loadRecentLogs(startTime, endTime time.Time, limit int) ([]types.DashboardRecentItem, error) {
	logs, err := s.caddyLogModel().Recent(s.ctx, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	items := make([]types.DashboardRecentItem, 0, len(logs))
	for _, logItem := range logs {
		items = append(items, types.DashboardRecentItem{
			ID:       logItem.ID,
			LogTime:  logItem.LogTime.Format("2006-01-02 15:04:05"),
			Method:   logItem.Method,
			Uri:      logItem.Uri,
			Status:   logItem.Status,
			RemoteIP: logItem.RemoteIP,
			Country:  logItem.Country,
		})
	}
	return items, nil
}

func (s *DashboardService) caddyLogModel() model.CaddyLogModel {
	if s.svcCtx.CaddyLogModel != nil {
		return s.svcCtx.CaddyLogModel
	}
	return model.NewCaddyLogModel(s.svcCtx.DB)
}
