// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"
	"fmt"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils"
	"logflux/internal/xerr"
	wafmodel "logflux/model/waf"

	"github.com/zeromicro/go-zero/core/logx"
)

// WAF 统计的默认时间窗口与分桶边界
const (
	wafStatsDefaultWindow = 24 * time.Hour
	wafStatsMinInterval   = 60
	wafStatsMaxInterval   = 86400
	wafStatsDefaultTopN   = 10
	wafStatsMaxTopN       = 50
)

type GetWafEventStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetWafEventStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWafEventStatsLogic {
	return &GetWafEventStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetWafEventStatsLogic) GetWafEventStats(req *types.WafEventStatsReq) (resp *types.WafEventStatsResp, err error) {
	start, end, err := l.resolveWindow(req)
	if err != nil {
		return nil, err
	}
	intervalSec := clampInterval(req.IntervalSec)
	topN := clampTopN(req.TopN)

	model := l.svcCtx.WafEventModel
	total, blocked, err := model.Count(l.ctx, start, end)
	if err != nil {
		l.Logger.Errorf("统计 WAF 事件失败: %v", err)
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计 WAF 事件失败", err)
	}
	uniqueIPs, err := model.CountUniqueIP(l.ctx, start, end)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计攻击源 IP 失败", err)
	}
	trendRows, err := model.TrendRows(l.ctx, start, end, intervalSec)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计 WAF 趋势失败", err)
	}
	topRuleRows, err := model.TopRules(l.ctx, start, end, topN)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计 Top 攻击规则失败", err)
	}
	topIPRows, err := model.TopIPs(l.ctx, start, end, topN)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计 Top 攻击 IP 失败", err)
	}
	topHostRows, err := model.TopHosts(l.ctx, start, end, topN)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计 Top 防护站点失败", err)
	}
	countryRows, err := model.CountryRows(l.ctx, start, end, topN)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计攻击源分布失败", err)
	}

	return &types.WafEventStatsResp{
		Total:     total,
		Blocked:   blocked,
		UniqueIPs: uniqueIPs,
		Trend:     buildWafTrendSeries(trendRows, start, end, intervalSec),
		TopRules:  toCountItems(topRuleRows),
		TopIPs:    toCountItems(topIPRows),
		TopHosts:  toCountItems(topHostRows),
		Countries: toGeoItems(countryRows),
	}, nil
}

// resolveWindow 解析统计时间窗口，缺省为最近 24 小时，起止倒置时自动交换。
func (l *GetWafEventStatsLogic) resolveWindow(req *types.WafEventStatsReq) (time.Time, time.Time, error) {
	start, err := utils.ParseOptionalTime(req.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, xerr.NewBusinessErrorWith(fmt.Sprintf("开始时间格式无效: %v", err))
	}
	end, err := utils.ParseOptionalTime(req.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, xerr.NewBusinessErrorWith(fmt.Sprintf("结束时间格式无效: %v", err))
	}

	now := time.Now()
	if end == nil {
		end = &now
	}
	if start == nil {
		defaultStart := end.Add(-wafStatsDefaultWindow)
		start = &defaultStart
	}
	if start.After(*end) {
		start, end = end, start
	}
	return *start, *end, nil
}

func clampInterval(intervalSec int) int {
	if intervalSec <= 0 {
		return 300
	}
	if intervalSec < wafStatsMinInterval {
		return wafStatsMinInterval
	}
	if intervalSec > wafStatsMaxInterval {
		return wafStatsMaxInterval
	}
	return intervalSec
}

func clampTopN(topN int) int {
	if topN <= 0 {
		return wafStatsDefaultTopN
	}
	if topN > wafStatsMaxTopN {
		return wafStatsMaxTopN
	}
	return topN
}

// buildWafTrendSeries 把趋势桶行转为补零等距序列，Total 含拦截与检测，Blocked 仅拦截。
func buildWafTrendSeries(rows []wafmodel.WafEventTrendRow, start, end time.Time, intervalSec int) []types.WafEventTrendItem {
	type bucketCount struct {
		total   int64
		blocked int64
	}
	buckets := make(map[int64]bucketCount, len(rows))
	for _, row := range rows {
		bc := buckets[row.Bucket]
		bc.total += row.Count
		if row.Interrupted {
			bc.blocked += row.Count
		}
		buckets[row.Bucket] = bc
	}

	bucketStart := (start.Unix() / int64(intervalSec)) * int64(intervalSec)
	bucketEnd := (end.Unix() / int64(intervalSec)) * int64(intervalSec)
	labelLayout := "15:04"
	if end.Sub(start) > 24*time.Hour {
		labelLayout = "01-02 15:04"
	}

	series := make([]types.WafEventTrendItem, 0)
	for ts := bucketStart; ts <= bucketEnd; ts += int64(intervalSec) {
		bc := buckets[ts]
		series = append(series, types.WafEventTrendItem{
			Time:    time.Unix(ts, 0).Format(labelLayout),
			Total:   bc.total,
			Blocked: bc.blocked,
		})
	}
	return series
}

func toCountItems(rows []wafmodel.WafEventCountRow) []types.WafEventCountItem {
	items := make([]types.WafEventCountItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.WafEventCountItem{Name: row.Name, Count: row.Count})
	}
	return items
}

func toGeoItems(rows []wafmodel.WafEventGeoRow) []types.WafEventGeoItem {
	items := make([]types.WafEventGeoItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.WafEventGeoItem{Name: row.Name, Value: row.Value})
	}
	return items
}
