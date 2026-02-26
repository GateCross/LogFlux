package caddy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetWafPolicyStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetWafPolicyStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWafPolicyStatsLogic {
	return &GetWafPolicyStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetWafPolicyStatsLogic) GetWafPolicyStats(req *types.WafPolicyStatsReq) (resp *types.WafPolicyStatsResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	startTime, endTime, intervalSec, err := normalizeWafPolicyStatsRange(req)
	if err != nil {
		return nil, err
	}
	topN := normalizeWafPolicyStatsTopN(req)
	drillFilter, err := normalizeWafPolicyStatsDrillFilter(req)
	if err != nil {
		return nil, err
	}

	policyQuery := l.svcCtx.DB.Model(&model.WafPolicy{}).Where("enabled = ?", true)
	if req != nil && req.PolicyId > 0 {
		policyQuery = policyQuery.Where("id = ?", req.PolicyId)
	}

	var policies []model.WafPolicy
	if err := policyQuery.Order("is_default desc, id asc").Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("query policy stats policies failed: %w", err)
	}

	if len(policies) == 0 {
		emptyTrend := buildEmptyWafPolicyStatsTrend(startTime, endTime, intervalSec)
		return &types.WafPolicyStatsResp{
			Range: types.DashboardRange{
				StartTime:   formatTime(startTime),
				EndTime:     formatTime(endTime),
				IntervalSec: intervalSec,
			},
			Summary: types.WafPolicyStatsItem{
				PolicyId:   0,
				PolicyName: "全部策略",
			},
			List:       []types.WafPolicyStatsItem{},
			Trend:      emptyTrend,
			TopHosts:   []types.WafPolicyStatsDimensionItem{},
			TopPaths:   []types.WafPolicyStatsDimensionItem{},
			TopMethods: []types.WafPolicyStatsDimensionItem{},
		}, nil
	}

	policyIDs := make([]uint, 0, len(policies))
	for _, policy := range policies {
		policyIDs = append(policyIDs, policy.ID)
	}

	var bindings []model.WafPolicyBinding
	if err := l.svcCtx.DB.
		Where("enabled = ? AND policy_id IN ?", true, policyIDs).
		Order("priority asc, id asc").
		Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("query policy stats bindings failed: %w", err)
	}

	bindingMap := make(map[uint][]model.WafPolicyBinding, len(policies))
	for _, binding := range bindings {
		bindingMap[binding.PolicyID] = append(bindingMap[binding.PolicyID], binding)
	}

	items := make([]types.WafPolicyStatsItem, 0, len(policies))
	for _, policy := range policies {
		item, itemErr := l.queryPolicyStatsItem(startTime, endTime, &policy, bindingMap[policy.ID], drillFilter)
		if itemErr != nil {
			return nil, itemErr
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].BlockedCount != items[j].BlockedCount {
			return items[i].BlockedCount > items[j].BlockedCount
		}
		if items[i].HitCount != items[j].HitCount {
			return items[i].HitCount > items[j].HitCount
		}
		return items[i].PolicyId < items[j].PolicyId
	})

	summary := types.WafPolicyStatsItem{
		PolicyId:   0,
		PolicyName: "全部策略",
	}
	trendBindings := []model.WafPolicyBinding(nil)
	trendAllLogs := true

	if req != nil && req.PolicyId > 0 {
		for _, item := range items {
			if item.PolicyId == req.PolicyId {
				summary = item
				break
			}
		}
		trendBindings = bindingMap[req.PolicyId]
		trendAllLogs = false
	} else {
		rangeSummary, sumErr := l.queryRangeSummary(startTime, endTime, drillFilter)
		if sumErr != nil {
			return nil, sumErr
		}
		summary = rangeSummary
	}

	trend, err := l.queryWafPolicyTrend(startTime, endTime, intervalSec, trendBindings, trendAllLogs, drillFilter)
	if err != nil {
		return nil, err
	}
	topHosts, topPaths, topMethods, err := l.queryWafPolicyDimensions(startTime, endTime, trendBindings, trendAllLogs, topN, drillFilter)
	if err != nil {
		return nil, err
	}

	return &types.WafPolicyStatsResp{
		Range: types.DashboardRange{
			StartTime:   formatTime(startTime),
			EndTime:     formatTime(endTime),
			IntervalSec: intervalSec,
		},
		Summary:    summary,
		List:       items,
		Trend:      trend,
		TopHosts:   topHosts,
		TopPaths:   topPaths,
		TopMethods: topMethods,
	}, nil
}

const wafPolicyStatsBlockedStatusSQL = "403,406,429"

var wafPolicyStatsBlockedStatuses = []int{403, 406, 429}

const (
	wafPolicyStatsDimensionEmptyHost = "(empty)"
	wafPolicyStatsDimensionPathRoot  = "/"
	wafPolicyStatsDimensionUnknown   = "UNKNOWN"
)

func normalizeWafPolicyStatsRange(req *types.WafPolicyStatsReq) (time.Time, time.Time, int, error) {
	var (
		start *time.Time
		end   *time.Time
		err   error
	)
	if req != nil {
		start, err = utils.ParseOptionalTime(req.StartTime)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid startTime: %w", err)
		}
		end, err = utils.ParseOptionalTime(req.EndTime)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid endTime: %w", err)
		}
	}

	now := time.Now()
	if end == nil {
		end = &now
	}
	if start == nil {
		defaultStart := end.Add(-24 * time.Hour)
		start = &defaultStart
	}
	if start.After(*end) {
		start, end = end, start
	}

	intervalSec := 300
	if req != nil && req.IntervalSec > 0 {
		intervalSec = req.IntervalSec
	}
	if intervalSec < 60 {
		intervalSec = 60
	}
	if intervalSec > 86400 {
		intervalSec = 86400
	}

	return *start, *end, intervalSec, nil
}

func normalizeWafPolicyStatsTopN(req *types.WafPolicyStatsReq) int {
	if req == nil || req.TopN <= 0 {
		return 8
	}
	if req.TopN < 1 {
		return 1
	}
	if req.TopN > 50 {
		return 50
	}
	return req.TopN
}

type wafPolicyStatsDrillFilter struct {
	Host   string
	Path   string
	Method string
}

func normalizeWafPolicyStatsDrillFilter(req *types.WafPolicyStatsReq) (wafPolicyStatsDrillFilter, error) {
	filter := wafPolicyStatsDrillFilter{}
	if req == nil {
		return filter, nil
	}

	host := strings.TrimSpace(req.Host)
	if host != "" {
		host = strings.ToLower(host)
		filter.Host = host
	}

	path := strings.TrimSpace(req.Path)
	if path != "" {
		if path != wafPolicyStatsDimensionPathRoot {
			path = normalizePolicyScopePath(path)
		}
		filter.Path = path
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method != "" {
		if method != wafPolicyStatsDimensionUnknown {
			if err := validatePolicyHTTPMethod(method); err != nil {
				return filter, err
			}
		}
		filter.Method = method
	}

	return filter, nil
}

func applyWafPolicyStatsDrillFilter(db *gorm.DB, filter wafPolicyStatsDrillFilter) *gorm.DB {
	if db == nil {
		return nil
	}

	if filter.Host != "" {
		if filter.Host == wafPolicyStatsDimensionEmptyHost {
			db = db.Where("COALESCE(NULLIF(TRIM(host), ''), ?) = ?", wafPolicyStatsDimensionEmptyHost, wafPolicyStatsDimensionEmptyHost)
		} else {
			db = db.Where("LOWER(host) = ?", filter.Host)
		}
	}

	if filter.Path != "" {
		db = db.Where("COALESCE(NULLIF(split_part(uri, chr(63), 1), ''), ?) = ?", wafPolicyStatsDimensionPathRoot, filter.Path)
	}

	if filter.Method != "" {
		if filter.Method == wafPolicyStatsDimensionUnknown {
			db = db.Where("COALESCE(NULLIF(UPPER(TRIM(method)), ''), ?) = ?", wafPolicyStatsDimensionUnknown, wafPolicyStatsDimensionUnknown)
		} else {
			db = db.Where("UPPER(method) = ?", filter.Method)
		}
	}

	return db
}

func (l *GetWafPolicyStatsLogic) queryPolicyStatsItem(
	startTime, endTime time.Time,
	policy *model.WafPolicy,
	bindings []model.WafPolicyBinding,
	drillFilter wafPolicyStatsDrillFilter,
) (types.WafPolicyStatsItem, error) {
	item := types.WafPolicyStatsItem{
		PolicyId:   policy.ID,
		PolicyName: strings.TrimSpace(policy.Name),
	}
	if item.PolicyName == "" {
		item.PolicyName = fmt.Sprintf("#%d", policy.ID)
	}

	base := l.svcCtx.DB.Model(&model.CaddyLog{}).Where("log_time BETWEEN ? AND ?", startTime, endTime)
	base = applyWafPolicyStatsDrillFilter(base, drillFilter)
	scoped := applyWafPolicyBindingScopeQuery(base, bindings)

	var hitCount int64
	if err := scoped.Count(&hitCount).Error; err != nil {
		return item, fmt.Errorf("count policy stats hits failed: %w", err)
	}
	item.HitCount = hitCount
	if hitCount <= 0 {
		return item, nil
	}

	var blockedCount int64
	if err := scoped.Where("status IN ?", wafPolicyStatsBlockedStatuses).Count(&blockedCount).Error; err != nil {
		return item, fmt.Errorf("count policy stats blocked hits failed: %w", err)
	}
	item.BlockedCount = blockedCount
	item.AllowedCount = hitCount - blockedCount
	item.BlockRate = calcPolicyBlockRate(item.BlockedCount, item.HitCount)

	suspectedCount, err := countWafPolicySuspectedFalsePositives(scoped)
	if err != nil {
		return item, err
	}
	item.SuspectedFalsePositiveCount = suspectedCount

	return item, nil
}

func (l *GetWafPolicyStatsLogic) queryRangeSummary(startTime, endTime time.Time, drillFilter wafPolicyStatsDrillFilter) (types.WafPolicyStatsItem, error) {
	summary := types.WafPolicyStatsItem{
		PolicyId:   0,
		PolicyName: "全部策略",
	}

	base := l.svcCtx.DB.Model(&model.CaddyLog{}).Where("log_time BETWEEN ? AND ?", startTime, endTime)
	base = applyWafPolicyStatsDrillFilter(base, drillFilter)

	var hitCount int64
	if err := base.Count(&hitCount).Error; err != nil {
		return summary, fmt.Errorf("count policy stats range hits failed: %w", err)
	}
	summary.HitCount = hitCount
	if hitCount <= 0 {
		return summary, nil
	}

	var blockedCount int64
	if err := base.Where("status IN ?", wafPolicyStatsBlockedStatuses).Count(&blockedCount).Error; err != nil {
		return summary, fmt.Errorf("count policy stats range blocked hits failed: %w", err)
	}
	summary.BlockedCount = blockedCount
	summary.AllowedCount = hitCount - blockedCount
	summary.BlockRate = calcPolicyBlockRate(summary.BlockedCount, summary.HitCount)

	suspectedCount, err := countWafPolicySuspectedFalsePositives(base)
	if err != nil {
		return summary, err
	}
	summary.SuspectedFalsePositiveCount = suspectedCount

	return summary, nil
}

func (l *GetWafPolicyStatsLogic) queryWafPolicyTrend(
	startTime, endTime time.Time,
	intervalSec int,
	bindings []model.WafPolicyBinding,
	allLogs bool,
	drillFilter wafPolicyStatsDrillFilter,
) ([]types.WafPolicyStatsTrendItem, error) {
	if !allLogs && len(bindings) == 0 {
		return buildEmptyWafPolicyStatsTrend(startTime, endTime, intervalSec), nil
	}

	type trendRow struct {
		Bucket       int64 `gorm:"column:bucket"`
		HitCount     int64 `gorm:"column:hit_count"`
		BlockedCount int64 `gorm:"column:blocked_count"`
	}

	db := l.svcCtx.DB.Model(&model.CaddyLog{}).Where("log_time BETWEEN ? AND ?", startTime, endTime)
	db = applyWafPolicyStatsDrillFilter(db, drillFilter)
	if !allLogs {
		db = applyWafPolicyBindingScopeQuery(db, bindings)
	}

	var rows []trendRow
	if err := db.
		Select(
			fmt.Sprintf(
				"floor(extract(epoch from log_time) / %d) * %d AS bucket, COUNT(*) AS hit_count, COALESCE(SUM(CASE WHEN status IN (%s) THEN 1 ELSE 0 END), 0) AS blocked_count",
				intervalSec, intervalSec, wafPolicyStatsBlockedStatusSQL,
			),
		).
		Group("bucket").
		Order("bucket").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query policy stats trend failed: %w", err)
	}

	bucketMap := make(map[int64]trendRow, len(rows))
	for _, row := range rows {
		bucketMap[row.Bucket] = row
	}

	startBucket := (startTime.Unix() / int64(intervalSec)) * int64(intervalSec)
	endBucket := (endTime.Unix() / int64(intervalSec)) * int64(intervalSec)
	series := make([]types.WafPolicyStatsTrendItem, 0, ((endBucket-startBucket)/int64(intervalSec))+1)
	for bucket := startBucket; bucket <= endBucket; bucket += int64(intervalSec) {
		row := bucketMap[bucket]
		series = append(series, types.WafPolicyStatsTrendItem{
			Time:         time.Unix(bucket, 0).Format("01-02 15:04"),
			HitCount:     row.HitCount,
			BlockedCount: row.BlockedCount,
			AllowedCount: row.HitCount - row.BlockedCount,
		})
	}

	return series, nil
}

func (l *GetWafPolicyStatsLogic) queryWafPolicyDimensions(
	startTime, endTime time.Time,
	bindings []model.WafPolicyBinding,
	allLogs bool,
	topN int,
	drillFilter wafPolicyStatsDrillFilter,
) ([]types.WafPolicyStatsDimensionItem, []types.WafPolicyStatsDimensionItem, []types.WafPolicyStatsDimensionItem, error) {
	if !allLogs && len(bindings) == 0 {
		return []types.WafPolicyStatsDimensionItem{}, []types.WafPolicyStatsDimensionItem{}, []types.WafPolicyStatsDimensionItem{}, nil
	}

	base := l.svcCtx.DB.Model(&model.CaddyLog{}).Where("log_time BETWEEN ? AND ?", startTime, endTime)
	base = applyWafPolicyStatsDrillFilter(base, drillFilter)
	if !allLogs {
		base = applyWafPolicyBindingScopeQuery(base, bindings)
	}

	topHosts, err := queryWafPolicyStatsDimension(base, "COALESCE(NULLIF(TRIM(host), ''), '(empty)')", topN, func(raw string) string {
		return normalizeWafPolicyDimensionKey(raw, wafPolicyStatsDimensionEmptyHost)
	})
	if err != nil {
		return nil, nil, nil, err
	}
	topPaths, err := queryWafPolicyStatsDimension(base, "COALESCE(NULLIF(split_part(uri, chr(63), 1), ''), '/')", topN, func(raw string) string {
		return normalizeWafPolicyDimensionKey(raw, wafPolicyStatsDimensionPathRoot)
	})
	if err != nil {
		return nil, nil, nil, err
	}
	topMethods, err := queryWafPolicyStatsDimension(base, "COALESCE(NULLIF(UPPER(TRIM(method)), ''), 'UNKNOWN')", topN, func(raw string) string {
		return normalizeWafPolicyDimensionKey(strings.ToUpper(strings.TrimSpace(raw)), wafPolicyStatsDimensionUnknown)
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return topHosts, topPaths, topMethods, nil
}

type wafPolicyDimensionRow struct {
	Key          string `gorm:"column:key"`
	HitCount     int64  `gorm:"column:hit_count"`
	BlockedCount int64  `gorm:"column:blocked_count"`
}

func queryWafPolicyStatsDimension(
	db *gorm.DB,
	keyExpr string,
	topN int,
	normalizeKey func(string) string,
) ([]types.WafPolicyStatsDimensionItem, error) {
	rows := make([]wafPolicyDimensionRow, 0)
	if err := db.
		Select(fmt.Sprintf(
			"%s AS key, COUNT(*) AS hit_count, COALESCE(SUM(CASE WHEN status IN (%s) THEN 1 ELSE 0 END), 0) AS blocked_count",
			keyExpr, wafPolicyStatsBlockedStatusSQL,
		)).
		Group("key").
		Order("hit_count DESC, blocked_count DESC, key ASC").
		Limit(topN).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query policy stats dimensions failed: %w", err)
	}

	items := make([]types.WafPolicyStatsDimensionItem, 0, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row.Key)
		if normalizeKey != nil {
			key = normalizeKey(key)
		}
		allowedCount := row.HitCount - row.BlockedCount
		items = append(items, types.WafPolicyStatsDimensionItem{
			Key:          key,
			HitCount:     row.HitCount,
			BlockedCount: row.BlockedCount,
			AllowedCount: allowedCount,
			BlockRate:    calcPolicyBlockRate(row.BlockedCount, row.HitCount),
		})
	}
	return items, nil
}

func normalizeWafPolicyDimensionKey(raw string, fallback string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		return "-"
	}
	return value
}

func applyWafPolicyBindingScopeQuery(db *gorm.DB, bindings []model.WafPolicyBinding) *gorm.DB {
	if db == nil {
		return nil
	}
	if len(bindings) == 0 {
		return db.Where("1 = 0")
	}

	for _, binding := range bindings {
		if normalizePolicyScopeType(binding.ScopeType) == wafPolicyScopeTypeGlobal {
			return db
		}
	}

	clauses := make([]string, 0, len(bindings))
	queryArgs := make([]interface{}, 0, len(bindings)*4)
	for _, binding := range bindings {
		condition, conditionArgs := buildWafPolicyBindingCondition(&binding)
		if condition == "" {
			continue
		}
		clauses = append(clauses, condition)
		queryArgs = append(queryArgs, conditionArgs...)
	}
	if len(clauses) == 0 {
		return db.Where("1 = 0")
	}
	return db.Where("("+strings.Join(clauses, " OR ")+")", queryArgs...)
}

func buildWafPolicyBindingCondition(binding *model.WafPolicyBinding) (string, []interface{}) {
	if binding == nil {
		return "", nil
	}

	scopeType := normalizePolicyScopeType(binding.ScopeType)
	switch scopeType {
	case wafPolicyScopeTypeGlobal:
		return "", nil
	case wafPolicyScopeTypeSite:
		host := normalizePolicyScopeHost(binding.Host)
		if host == "" {
			return "", nil
		}
		return "LOWER(host) = ?", []interface{}{host}
	case wafPolicyScopeTypeRoute:
		parts := make([]string, 0, 3)
		args := make([]interface{}, 0, 6)

		host := normalizePolicyScopeHost(binding.Host)
		if host != "" {
			parts = append(parts, "LOWER(host) = ?")
			args = append(args, host)
		}

		path := normalizePolicyScopePath(binding.Path)
		pathCondition, pathArgs := buildWafPolicyPathMatchCondition(path)
		if pathCondition != "" {
			parts = append(parts, pathCondition)
			args = append(args, pathArgs...)
		}

		method := normalizePolicyHTTPMethod(binding.Method)
		if method != "" {
			parts = append(parts, "method = ?")
			args = append(args, method)
		}

		if len(parts) == 0 {
			return "", nil
		}
		return "(" + strings.Join(parts, " AND ") + ")", args
	default:
		return "", nil
	}
}

func buildWafPolicyPathMatchCondition(path string) (string, []interface{}) {
	normalized := normalizePolicyScopePath(path)
	if normalized == "" {
		return "", nil
	}
	if normalized == "/" {
		return "uri LIKE '/%'", nil
	}
	return "(uri = ? OR uri LIKE ? OR uri LIKE ?)", []interface{}{normalized, normalized + "/%", normalized + "?%"}
}

func countWafPolicySuspectedFalsePositives(query *gorm.DB) (int64, error) {
	if query == nil {
		return 0, nil
	}

	paths := []string{"/health", "/healthz", "/ready", "/live", "/status", "/metrics", "/favicon.ico", "/robots.txt"}
	pathConditions := make([]string, 0, len(paths))
	pathArgs := make([]interface{}, 0, len(paths)*3)
	for _, path := range paths {
		condition, args := buildWafPolicyPathMatchCondition(path)
		if condition == "" {
			continue
		}
		pathConditions = append(pathConditions, condition)
		pathArgs = append(pathArgs, args...)
	}
	if len(pathConditions) == 0 {
		return 0, nil
	}

	heuristicQuery := query.
		Where("status IN ?", wafPolicyStatsBlockedStatuses).
		Where("method IN ?", []string{"GET", "HEAD", "OPTIONS"}).
		Where("("+strings.Join(pathConditions, " OR ")+")", pathArgs...)

	var count int64
	if err := heuristicQuery.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count policy stats suspected false positives failed: %w", err)
	}
	return count, nil
}

func buildEmptyWafPolicyStatsTrend(startTime, endTime time.Time, intervalSec int) []types.WafPolicyStatsTrendItem {
	startBucket := (startTime.Unix() / int64(intervalSec)) * int64(intervalSec)
	endBucket := (endTime.Unix() / int64(intervalSec)) * int64(intervalSec)

	series := make([]types.WafPolicyStatsTrendItem, 0, ((endBucket-startBucket)/int64(intervalSec))+1)
	for bucket := startBucket; bucket <= endBucket; bucket += int64(intervalSec) {
		series = append(series, types.WafPolicyStatsTrendItem{
			Time:         time.Unix(bucket, 0).Format("01-02 15:04"),
			HitCount:     0,
			BlockedCount: 0,
			AllowedCount: 0,
		})
	}
	return series
}

func calcPolicyBlockRate(blocked, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(blocked) / float64(total)
}
