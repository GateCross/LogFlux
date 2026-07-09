package service

import (
	"context"
	"fmt"
	caddymodel "logflux/model/caddy"
	ingestmodel "logflux/model/ingest"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"
)

// 站点近窗指标约束（与 design：hosts 上限、默认近窗一致）
const (
	siteMetricsMaxHosts             = 50
	siteMetricsDefaultWindowMinutes = 15
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

	logs, total, err := s.caddyLogModel().List(s.ctx, caddymodel.CaddyLogQuery{
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
		country, province, city := s.resolveCaddyLogRegion(logItem)
		location := strings.TrimSpace(strings.Join([]string{
			strings.TrimSpace(country),
			strings.TrimSpace(province),
			strings.TrimSpace(city),
		}, " "))
		list = append(list, types.CaddyLogItem{
			ID:        logItem.ID,
			LogTime:   logItem.LogTime.Format("2006-01-02 15:04:05"),
			Country:   country,
			Province:  province,
			City:      city,
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

// 按 host 批量返回近窗 4xx/5xx 计数。
// 无日志 host 补 0，不因单 host 无数据失败整批。
func (s *LogService) GetSiteMetrics(req *types.SiteMetricsReq) (*types.SiteMetricsResp, error) {
	if req == nil {
		return nil, xerr.NewBusinessErrorWith("请求参数无效")
	}

	hosts := normalizeSiteMetricHosts(req.Hosts)
	if len(hosts) == 0 {
		return nil, xerr.NewBusinessErrorWith("hosts 不能为空")
	}
	if len(hosts) > siteMetricsMaxHosts {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("hosts 数量不能超过 %d", siteMetricsMaxHosts))
	}

	windowMinutes := req.WindowMinutes
	if windowMinutes <= 0 {
		windowMinutes = siteMetricsDefaultWindowMinutes
	}

	end := time.Now()
	start := end.Add(-time.Duration(windowMinutes) * time.Minute)

	logModel := s.caddyLogModel()
	rows4xx, err := logModel.CountStatusByHosts(s.ctx, start, end, hosts, 400, 500)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计站点 4xx 失败", err)
	}
	rows5xx, err := logModel.CountStatusByHosts(s.ctx, start, end, hosts, 500, 600)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "统计站点 5xx 失败", err)
	}

	return &types.SiteMetricsResp{
		List: mergeHostStatusCounts(hosts, rows4xx, rows5xx),
	}, nil
}

// 将 model 返回的稀疏计数合并为请求 hosts 的完整列表，缺失补 0。
// 导出给测试用于验证 Property 6 的聚合语义。
func mergeHostStatusCounts(hosts []string, rows4xx, rows5xx []caddymodel.HostStatusCount) []types.SiteMetricsItem {
	count4xx := hostCountMap(rows4xx)
	count5xx := hostCountMap(rows5xx)

	list := make([]types.SiteMetricsItem, 0, len(hosts))
	for _, host := range hosts {
		list = append(list, types.SiteMetricsItem{
			Host:     host,
			Count4xx: count4xx[host],
			Count5xx: count5xx[host],
		})
	}
	return list
}

func hostCountMap(rows []caddymodel.HostStatusCount) map[string]int64 {
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Host] = row.Count
	}
	return out
}

// 去空白并去重，保持首次出现顺序。
func normalizeSiteMetricHosts(hosts []string) []string {
	if len(hosts) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(hosts))
	out := make([]string, 0, len(hosts))
	for _, h := range hosts {
		host := strings.TrimSpace(h)
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}
	return out
}

func (s *LogService) resolveCaddyLogRegion(logItem caddymodel.CaddyLog) (country, province, city string) {
	country = strings.TrimSpace(logItem.Country)
	province = strings.TrimSpace(logItem.Province)
	city = strings.TrimSpace(logItem.City)
	if country != "" || province != "" || city != "" || s.svcCtx.IPRegionMgr == nil {
		return country, province, city
	}

	ip := strings.TrimSpace(logItem.ClientIP)
	if ip == "" {
		ip = strings.TrimSpace(logItem.RemoteIP)
	}
	if ip == "" {
		return country, province, city
	}

	return s.svcCtx.IPRegionMgr.Resolve(ip)
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

	logs, total, err := s.systemLogModel().List(s.ctx, ingestmodel.SystemLogQuery{
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

func (s *LogService) ClearSystemLogs() (*types.BaseResp, error) {
	if err := s.systemLogModel().Clear(s.ctx); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "清空系统日志失败", err)
	}
	return &types.BaseResp{Code: 200, Msg: "成功"}, nil
}

func (s *LogService) ClearCaddyLogs() (*types.BaseResp, error) {
	if err := s.caddyLogModel().Clear(s.ctx); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "清空访问日志失败", err)
	}
	return &types.BaseResp{Code: 200, Msg: "成功"}, nil
}

func (s *LogService) caddyLogModel() caddymodel.CaddyLogModel {
	if s.svcCtx.CaddyLogModel != nil {
		return s.svcCtx.CaddyLogModel
	}
	return caddymodel.NewCaddyLogModel(s.svcCtx.DB)
}

func (s *LogService) systemLogModel() ingestmodel.SystemLogModel {
	if s.svcCtx.SystemLogModel != nil {
		return s.svcCtx.SystemLogModel
	}
	return ingestmodel.NewSystemLogModel(s.svcCtx.DB)
}
