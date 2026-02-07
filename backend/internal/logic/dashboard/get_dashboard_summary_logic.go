package dashboard

import (
	"context"
	"fmt"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetDashboardSummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDashboardSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDashboardSummaryLogic {
	return &GetDashboardSummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDashboardSummaryLogic) GetDashboardSummary(req *types.DashboardSummaryReq) (resp *types.DashboardSummaryResp, err error) {
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

	queryLogs := func() *gorm.DB {
		return l.svcCtx.DB.Model(&model.CaddyLog{}).Where("log_time >= ? AND log_time <= ?", startTime, endTime)
	}

	var total int64
	if err := queryLogs().Count(&total).Error; err != nil {
		return nil, err
	}

	var blocked int64
	blockedStatuses := []int{403, 429}
	if err := queryLogs().Where("status IN ?", blockedStatuses).Count(&blocked).Error; err != nil {
		return nil, err
	}

	var err4xx int64
	if err := queryLogs().Where("status >= ? AND status < ?", 400, 500).Count(&err4xx).Error; err != nil {
		return nil, err
	}

	var err5xx int64
	if err := queryLogs().Where("status >= ? AND status < ?", 500, 600).Count(&err5xx).Error; err != nil {
		return nil, err
	}

	var uv int64
	if err := l.svcCtx.DB.Raw(
		`SELECT COUNT(DISTINCT COALESCE(NULLIF(client_ip, ''), remote_ip))
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ? AND (client_ip <> '' OR remote_ip <> '')`,
		startTime, endTime,
	).Scan(&uv).Error; err != nil {
		return nil, err
	}

	var uniqueIP int64
	if err := l.svcCtx.DB.Raw(
		"SELECT COUNT(DISTINCT remote_ip) FROM caddy_logs WHERE log_time BETWEEN ? AND ? AND remote_ip <> ''",
		startTime, endTime,
	).Scan(&uniqueIP).Error; err != nil {
		return nil, err
	}

	var attackIP int64
	if err := l.svcCtx.DB.Raw(
		"SELECT COUNT(DISTINCT remote_ip) FROM caddy_logs WHERE log_time BETWEEN ? AND ? AND status IN ? AND remote_ip <> ''",
		startTime, endTime, blockedStatuses,
	).Scan(&attackIP).Error; err != nil {
		return nil, err
	}

	trend, err := loadTrendSeries(l.svcCtx.DB, startTime, endTime, intervalSec)
	if err != nil {
		return nil, err
	}

	geo, err := loadGeoStats(l.svcCtx.DB, startTime, endTime, topN)
	if err != nil {
		return nil, err
	}

	recent, err := loadRecentLogs(l.svcCtx.DB, startTime, endTime, recentLimit)
	if err != nil {
		return nil, err
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
		return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid startTime: %w", err)
	}
	end, err := utils.ParseOptionalTime(req.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid endTime: %w", err)
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

type trendRow struct {
	Bucket int64 `gorm:"column:bucket"`
	Count  int64 `gorm:"column:count"`
}

func loadTrendSeries(db *gorm.DB, startTime, endTime time.Time, intervalSec int) ([]types.DashboardTrendItem, error) {
	rows := make([]trendRow, 0)
	err := db.Raw(
		`SELECT floor(extract(epoch from log_time) / ?) * ? AS bucket, COUNT(*) AS count
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ?
		 GROUP BY bucket
		 ORDER BY bucket`,
		intervalSec, intervalSec, startTime, endTime,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	bucketMap := make(map[int64]int64, len(rows))
	for _, row := range rows {
		bucketMap[row.Bucket] = row.Count
	}

	startUnix := startTime.Unix()
	endUnix := endTime.Unix()
	bucketStart := (startUnix / int64(intervalSec)) * int64(intervalSec)
	bucketEnd := (endUnix / int64(intervalSec)) * int64(intervalSec)

	labelLayout := "15:04"
	if endTime.Sub(startTime) > 24*time.Hour {
		labelLayout = "01-02 15:04"
	}

	series := make([]types.DashboardTrendItem, 0)
	for ts := bucketStart; ts <= bucketEnd; ts += int64(intervalSec) {
		value := bucketMap[ts]
		series = append(series, types.DashboardTrendItem{
			Time:  time.Unix(ts, 0).Format(labelLayout),
			Value: value,
		})
	}

	return series, nil
}

type geoRow struct {
	Name  string `gorm:"column:name"`
	Value int64  `gorm:"column:value"`
}

func loadGeoStats(db *gorm.DB, startTime, endTime time.Time, topN int) ([]types.DashboardGeoItem, error) {
	rows := make([]geoRow, 0)
	err := db.Raw(
		`SELECT COALESCE(NULLIF(country, ''), '未知') AS name, COUNT(*) AS value
		 FROM caddy_logs
		 WHERE log_time BETWEEN ? AND ?
		 GROUP BY name
		 ORDER BY value DESC
		 LIMIT ?`,
		startTime, endTime, topN,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]types.DashboardGeoItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.DashboardGeoItem{
			Name:  row.Name,
			Value: row.Value,
		})
	}
	return items, nil
}

func loadRecentLogs(db *gorm.DB, startTime, endTime time.Time, limit int) ([]types.DashboardRecentItem, error) {
	var logs []model.CaddyLog
	if err := db.
		Where("log_time >= ? AND log_time <= ?", startTime, endTime).
		Order("log_time desc, id desc").
		Limit(limit).
		Find(&logs).Error; err != nil {
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
