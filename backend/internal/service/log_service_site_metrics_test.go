package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	caddymodel "logflux/model/caddy"
	wafmodel "logflux/model/waf"

	"logflux/internal/svc"
	"logflux/internal/types"
)

// fixtureLogRow 是 Property 6 预言机计数用的最小访问日志行。
type fixtureLogRow struct {
	Host   string
	Status int
	At     time.Time
}

// oracleHostCounts 按 host 统计时间窗 [start, end] 内总请求/4xx/5xx。
// 状态范围与 model 一致：4xx ∈ [400,500)，5xx ∈ [500,600)；总数不筛状态。
func oracleHostCounts(rows []fixtureLogRow, hosts []string, start, end time.Time) map[string][3]int64 {
	out := make(map[string][3]int64, len(hosts))
	for _, h := range hosts {
		out[h] = [3]int64{0, 0, 0}
	}
	for _, row := range rows {
		if row.At.Before(start) || row.At.After(end) {
			continue
		}
		counts, ok := out[row.Host]
		if !ok {
			continue
		}
		counts[0]++
		switch {
		case row.Status >= 400 && row.Status < 500:
			counts[1]++
		case row.Status >= 500 && row.Status < 600:
			counts[2]++
		}
		out[row.Host] = counts
	}
	return out
}

// 将预言机结果转为 model 风格的稀疏计数切片
// （无匹配行的 host 省略——与按 host 聚合接口契约一致）。
func rowsToSparse(oracle map[string][3]int64) (rowsTotal, rows4xx, rows5xx []caddymodel.HostStatusCount) {
	for host, counts := range oracle {
		if counts[0] > 0 {
			rowsTotal = append(rowsTotal, caddymodel.HostStatusCount{Host: host, Count: counts[0]})
		}
		if counts[1] > 0 {
			rows4xx = append(rows4xx, caddymodel.HostStatusCount{Host: host, Count: counts[1]})
		}
		if counts[2] > 0 {
			rows5xx = append(rows5xx, caddymodel.HostStatusCount{Host: host, Count: counts[2]})
		}
	}
	return rowsTotal, rows4xx, rows5xx
}

// Property 6: 近窗按 host 聚合
// 对任意 host 集合与时间窗，返回的 5xx（及 4xx）计数应等于该 host
// 在窗内对应状态范围的访问日志行数；无日志则为 0。
// **验证：Requirements 5.1, 5.3**
func TestProperty6_NearWindowHostAggregation(t *testing.T) {
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.Local)
	start := now.Add(-15 * time.Minute)
	end := now

	// 固定夹具：窗内/窗外、混合状态、零日志 host
	fixtures := []fixtureLogRow{
		{Host: "a.example.com", Status: 200, At: now.Add(-2 * time.Minute)},
		{Host: "a.example.com", Status: 404, At: now.Add(-3 * time.Minute)},
		{Host: "a.example.com", Status: 499, At: now.Add(-4 * time.Minute)},
		{Host: "a.example.com", Status: 500, At: now.Add(-5 * time.Minute)},
		{Host: "a.example.com", Status: 503, At: now.Add(-6 * time.Minute)},
		// 窗外 — 不得计入
		{Host: "a.example.com", Status: 500, At: now.Add(-30 * time.Minute)},
		{Host: "b.example.com", Status: 401, At: now.Add(-1 * time.Minute)},
		{Host: "b.example.com", Status: 502, At: now.Add(-1 * time.Minute)},
		// 未请求的 host
		{Host: "other.example.com", Status: 500, At: now.Add(-1 * time.Minute)},
		// 边界状态
		{Host: "c.example.com", Status: 399, At: now.Add(-1 * time.Minute)}, // 非 4xx
		{Host: "c.example.com", Status: 400, At: now.Add(-1 * time.Minute)},
		{Host: "c.example.com", Status: 599, At: now.Add(-1 * time.Minute)},
		{Host: "c.example.com", Status: 600, At: now.Add(-1 * time.Minute)}, // 非 5xx
	}

	cases := []struct {
		name  string
		hosts []string
	}{
		{
			name:  "mixed hosts with partial zeros",
			hosts: []string{"a.example.com", "b.example.com", "empty.example.com"},
		},
		{
			name:  "only empty host",
			hosts: []string{"no-logs.example.com"},
		},
		{
			name:  "boundary status host",
			hosts: []string{"c.example.com"},
		},
		{
			name:  "single host with 4xx and 5xx",
			hosts: []string{"a.example.com"},
		},
		{
			name:  "preserve request order with zeros",
			hosts: []string{"empty.example.com", "b.example.com", "a.example.com"},
		},
	}

	// 拦截稀疏输入：waf_events 按 host 聚合行的模拟，与访问日志计数相互独立
	blockedRows := []wafmodel.HostInterruptedCount{
		{Host: "a.example.com", Count: 5},
		{Host: "b.example.com", Count: 2},
	}
	blockedByHost := map[string]int64{"a.example.com": 5, "b.example.com": 2}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oracle := oracleHostCounts(fixtures, tc.hosts, start, end)
			rowsTotal, rows4xx, rows5xx := rowsToSparse(oracle)

			// Model 返回稀疏行；merge 须对请求的全部 host 补 0
			got := mergeHostMetrics(tc.hosts, rowsTotal, rows4xx, rows5xx, blockedRows)
			if len(got) != len(tc.hosts) {
				t.Fatalf("expected %d items, got %d", len(tc.hosts), len(got))
			}
			for i, host := range tc.hosts {
				if got[i].Host != host {
					t.Fatalf("order mismatch at %d: want host %q, got %q", i, host, got[i].Host)
				}
				wantTotal, want4, want5 := oracle[host][0], oracle[host][1], oracle[host][2]
				if got[i].Total != wantTotal || got[i].Count4xx != want4 || got[i].Count5xx != want5 {
					t.Fatalf("host %s: want total=%d 4xx=%d 5xx=%d, got total=%d 4xx=%d 5xx=%d",
						host, wantTotal, want4, want5, got[i].Total, got[i].Count4xx, got[i].Count5xx)
				}
				if got[i].Blocked != blockedByHost[host] {
					t.Fatalf("host %s: want blocked=%d, got %d", host, blockedByHost[host], got[i].Blocked)
				}
			}
		})
	}

	// 显式零填充要求（Req 5.3）
	t.Run("no logs for host returns zero not omit", func(t *testing.T) {
		hosts := []string{"ghost.example.com", "a.example.com"}
		oracle := oracleHostCounts(fixtures, hosts, start, end)
		rowsTotal, rows4xx, rows5xx := rowsToSparse(oracle)
		got := mergeHostMetrics(hosts, rowsTotal, rows4xx, rows5xx, blockedRows)
		if len(got) != 2 {
			t.Fatalf("expected 2 items including zero host, got %d", len(got))
		}
		if got[0].Host != "ghost.example.com" || got[0].Total != 0 || got[0].Count4xx != 0 || got[0].Count5xx != 0 || got[0].Blocked != 0 {
			t.Fatalf("expected zero-filled ghost host, got %+v", got[0])
		}
		if got[1].Total == 0 && got[1].Count4xx == 0 && got[1].Count5xx == 0 {
			t.Fatalf("expected a.example.com to have non-zero counts in fixture")
		}
	})
}

// 仅为站点指标服务测试实现按 host 状态聚合。
type mockCaddyLogModel struct {
	caddymodel.CaddyLogModel // 嵌入 nil 以满足未用方法
	totalFn                  func(ctx context.Context, start, end time.Time, hosts []string) ([]caddymodel.HostStatusCount, error)
	countFn                  func(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]caddymodel.HostStatusCount, error)
	calls                    []mockCountCall
	totalCalls               int
}

type mockCountCall struct {
	Hosts []string
	Min   int
	Max   int
}

func (m *mockCaddyLogModel) CountAllByHosts(ctx context.Context, start, end time.Time, hosts []string) ([]caddymodel.HostStatusCount, error) {
	m.totalCalls++
	if m.totalFn != nil {
		return m.totalFn(ctx, start, end, hosts)
	}
	return []caddymodel.HostStatusCount{}, nil
}

func (m *mockCaddyLogModel) CountStatusByHosts(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]caddymodel.HostStatusCount, error) {
	m.calls = append(m.calls, mockCountCall{Hosts: append([]string(nil), hosts...), Min: min, Max: max})
	if m.countFn != nil {
		return m.countFn(ctx, start, end, hosts, min, max)
	}
	return []caddymodel.HostStatusCount{}, nil
}

// 仅为站点指标服务测试实现按 host 拦截聚合。
type mockWafEventModel struct {
	wafmodel.WafEventModel // 嵌入 nil 以满足未用方法
	blockedFn              func(ctx context.Context, start, end time.Time, hosts []string) ([]wafmodel.HostInterruptedCount, error)
	blockedCalls           int
}

func (m *mockWafEventModel) CountInterruptedByHosts(ctx context.Context, start, end time.Time, hosts []string) ([]wafmodel.HostInterruptedCount, error) {
	m.blockedCalls++
	if m.blockedFn != nil {
		return m.blockedFn(ctx, start, end, hosts)
	}
	return []wafmodel.HostInterruptedCount{}, nil
}

// 未使用的接口方法若被误调用则 panic。
func (m *mockCaddyLogModel) Create(context.Context, *caddymodel.CaddyLog) error {
	panic("unexpected Create")
}
func (m *mockCaddyLogModel) List(context.Context, caddymodel.CaddyLogQuery) ([]caddymodel.CaddyLog, int64, error) {
	panic("unexpected List")
}
func (m *mockCaddyLogModel) Clear(context.Context) error { panic("unexpected Clear") }
func (m *mockCaddyLogModel) CountRange(context.Context, time.Time, time.Time) (int64, error) {
	panic("unexpected CountRange")
}
func (m *mockCaddyLogModel) CountStatuses(context.Context, time.Time, time.Time, []int) (int64, error) {
	panic("unexpected CountStatuses")
}
func (m *mockCaddyLogModel) CountStatusRange(context.Context, time.Time, time.Time, int, int) (int64, error) {
	panic("unexpected CountStatusRange")
}
func (m *mockCaddyLogModel) CountUniqueVisitor(context.Context, time.Time, time.Time) (int64, error) {
	panic("unexpected CountUniqueVisitor")
}
func (m *mockCaddyLogModel) CountUniqueRemoteIP(context.Context, time.Time, time.Time) (int64, error) {
	panic("unexpected CountUniqueRemoteIP")
}
func (m *mockCaddyLogModel) CountAttackIP(context.Context, time.Time, time.Time, []int) (int64, error) {
	panic("unexpected CountAttackIP")
}
func (m *mockCaddyLogModel) TrendRows(context.Context, time.Time, time.Time, int) ([]caddymodel.DashboardTrendRow, error) {
	panic("unexpected TrendRows")
}
func (m *mockCaddyLogModel) GeoRows(context.Context, time.Time, time.Time, int) ([]caddymodel.DashboardGeoRow, error) {
	panic("unexpected GeoRows")
}
func (m *mockCaddyLogModel) ProvinceRows(context.Context, time.Time, time.Time, int) ([]caddymodel.DashboardGeoRow, error) {
	panic("unexpected ProvinceRows")
}
func (m *mockCaddyLogModel) Recent(context.Context, time.Time, time.Time, int) ([]caddymodel.CaddyLog, error) {
	panic("unexpected Recent")
}

func TestGetSiteMetrics_ServiceZeroFillAndRanges(t *testing.T) {
	mock := &mockCaddyLogModel{
		totalFn: func(ctx context.Context, start, end time.Time, hosts []string) ([]caddymodel.HostStatusCount, error) {
			// 稀疏：仅 a.example.com 有请求；empty.example.com 省略
			return []caddymodel.HostStatusCount{{Host: "a.example.com", Count: 10}}, nil
		},
		countFn: func(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]caddymodel.HostStatusCount, error) {
			// 稀疏：仅 a.example.com 有数据；empty.example.com 省略
			if min == 400 && max == 500 {
				return []caddymodel.HostStatusCount{{Host: "a.example.com", Count: 2}}, nil
			}
			if min == 500 && max == 600 {
				return []caddymodel.HostStatusCount{{Host: "a.example.com", Count: 3}}, nil
			}
			return nil, fmt.Errorf("unexpected range %d-%d", min, max)
		},
	}
	wafMock := &mockWafEventModel{
		blockedFn: func(ctx context.Context, start, end time.Time, hosts []string) ([]wafmodel.HostInterruptedCount, error) {
			return []wafmodel.HostInterruptedCount{{Host: "a.example.com", Count: 4}}, nil
		},
	}
	svcCtx := &svc.ServiceContext{CaddyLogModel: mock, WafEventModel: wafMock}
	logSvc := NewLogService(context.Background(), svcCtx)

	resp, err := logSvc.GetSiteMetrics(&types.SiteMetricsReq{
		Hosts:         []string{"a.example.com", "empty.example.com"},
		WindowMinutes: 15,
	})
	if err != nil {
		t.Fatalf("GetSiteMetrics error: %v", err)
	}
	if len(resp.List) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(resp.List))
	}
	if resp.List[0].Host != "a.example.com" || resp.List[0].Total != 10 ||
		resp.List[0].Count4xx != 2 || resp.List[0].Count5xx != 3 || resp.List[0].Blocked != 4 {
		t.Fatalf("unexpected a.example.com metrics: %+v", resp.List[0])
	}
	if resp.List[1].Host != "empty.example.com" || resp.List[1].Total != 0 ||
		resp.List[1].Count4xx != 0 || resp.List[1].Count5xx != 0 || resp.List[1].Blocked != 0 {
		t.Fatalf("expected zero-filled empty host, got %+v", resp.List[1])
	}

	// 必须调用 model 两次：先 4xx 再 5xx；总请求与拦截各一次
	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 CountStatusByHosts calls, got %d", len(mock.calls))
	}
	if mock.calls[0].Min != 400 || mock.calls[0].Max != 500 {
		t.Fatalf("first call should be 4xx range, got %+v", mock.calls[0])
	}
	if mock.calls[1].Min != 500 || mock.calls[1].Max != 600 {
		t.Fatalf("second call should be 5xx range, got %+v", mock.calls[1])
	}
	if mock.totalCalls != 1 {
		t.Fatalf("expected 1 CountAllByHosts call, got %d", mock.totalCalls)
	}
	if wafMock.blockedCalls != 1 {
		t.Fatalf("expected 1 CountInterruptedByHosts call, got %d", wafMock.blockedCalls)
	}
}

func TestGetSiteMetrics_Validation(t *testing.T) {
	logSvc := NewLogService(context.Background(), &svc.ServiceContext{
		CaddyLogModel: &mockCaddyLogModel{},
		WafEventModel: &mockWafEventModel{},
	})

	t.Run("empty hosts", func(t *testing.T) {
		_, err := logSvc.GetSiteMetrics(&types.SiteMetricsReq{Hosts: nil})
		if err == nil || !strings.Contains(err.Error(), "hosts 不能为空") {
			t.Fatalf("expected empty hosts error, got %v", err)
		}
	})

	t.Run("blank hosts only", func(t *testing.T) {
		_, err := logSvc.GetSiteMetrics(&types.SiteMetricsReq{Hosts: []string{" ", ""}})
		if err == nil || !strings.Contains(err.Error(), "hosts 不能为空") {
			t.Fatalf("expected empty hosts error, got %v", err)
		}
	})

	t.Run("hosts over cap", func(t *testing.T) {
		hosts := make([]string, siteMetricsMaxHosts+1)
		for i := range hosts {
			hosts[i] = fmt.Sprintf("h%d.example.com", i)
		}
		_, err := logSvc.GetSiteMetrics(&types.SiteMetricsReq{Hosts: hosts})
		if err == nil || !strings.Contains(err.Error(), "hosts 数量不能超过") {
			t.Fatalf("expected hosts cap error, got %v", err)
		}
	})

	t.Run("default window when omitted", func(t *testing.T) {
		var gotWindow time.Duration
		mock := &mockCaddyLogModel{
			countFn: func(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]caddymodel.HostStatusCount, error) {
				gotWindow = end.Sub(start)
				return nil, nil
			},
		}
		logSvc := NewLogService(context.Background(), &svc.ServiceContext{CaddyLogModel: mock, WafEventModel: &mockWafEventModel{}})
		_, err := logSvc.GetSiteMetrics(&types.SiteMetricsReq{Hosts: []string{"a.example.com"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 允许少量时钟偏差；窗口应约 15 分钟
		want := time.Duration(siteMetricsDefaultWindowMinutes) * time.Minute
		if gotWindow < want-time.Second || gotWindow > want+time.Second {
			t.Fatalf("expected ~%v window, got %v", want, gotWindow)
		}
	})

	// 查询窗口切换：<=0 回落默认窗，1440 即 24h
	t.Run("window override", func(t *testing.T) {
		cases := []struct {
			name          string
			windowMinutes int
			want          time.Duration
		}{
			{"zero falls back to default", 0, time.Duration(siteMetricsDefaultWindowMinutes) * time.Minute},
			{"negative falls back to default", -1, time.Duration(siteMetricsDefaultWindowMinutes) * time.Minute},
			{"24h window", 1440, 1440 * time.Minute},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var gotWindow time.Duration
				mock := &mockCaddyLogModel{
					countFn: func(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]caddymodel.HostStatusCount, error) {
						gotWindow = end.Sub(start)
						return nil, nil
					},
				}
				logSvc := NewLogService(context.Background(), &svc.ServiceContext{CaddyLogModel: mock, WafEventModel: &mockWafEventModel{}})
				_, err := logSvc.GetSiteMetrics(&types.SiteMetricsReq{
					Hosts:         []string{"a.example.com"},
					WindowMinutes: tc.windowMinutes,
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if gotWindow < tc.want-time.Second || gotWindow > tc.want+time.Second {
					t.Fatalf("windowMinutes=%d: expected ~%v window, got %v", tc.windowMinutes, tc.want, gotWindow)
				}
			})
		}
	})

	t.Run("dedupe hosts", func(t *testing.T) {
		mock := &mockCaddyLogModel{
			countFn: func(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]caddymodel.HostStatusCount, error) {
				return []caddymodel.HostStatusCount{}, nil
			},
		}
		wafMock := &mockWafEventModel{}
		logSvc := NewLogService(context.Background(), &svc.ServiceContext{CaddyLogModel: mock, WafEventModel: wafMock})
		resp, err := logSvc.GetSiteMetrics(&types.SiteMetricsReq{
			Hosts: []string{"a.example.com", " a.example.com ", "b.example.com"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.List) != 2 {
			t.Fatalf("expected deduped 2 hosts, got %d: %+v", len(resp.List), resp.List)
		}
		if mock.calls[0].Hosts[0] != "a.example.com" || len(mock.calls[0].Hosts) != 2 {
			t.Fatalf("model should receive deduped hosts, got %+v", mock.calls[0].Hosts)
		}
	})
}

func TestGetSiteMetricsLogic_EndToEndWithMock(t *testing.T) {
	// 覆盖 handler → logic → service 的集成路径
	mock := &mockCaddyLogModel{
		totalFn: func(ctx context.Context, start, end time.Time, hosts []string) ([]caddymodel.HostStatusCount, error) {
			return []caddymodel.HostStatusCount{{Host: "api.example.com", Count: 20}}, nil
		},
		countFn: func(ctx context.Context, start, end time.Time, hosts []string, min, max int) ([]caddymodel.HostStatusCount, error) {
			if min == 500 {
				return []caddymodel.HostStatusCount{{Host: "api.example.com", Count: 7}}, nil
			}
			return []caddymodel.HostStatusCount{}, nil
		},
	}
	wafMock := &mockWafEventModel{
		blockedFn: func(ctx context.Context, start, end time.Time, hosts []string) ([]wafmodel.HostInterruptedCount, error) {
			return []wafmodel.HostInterruptedCount{{Host: "api.example.com", Count: 3}}, nil
		},
	}
	svcCtx := &svc.ServiceContext{CaddyLogModel: mock, WafEventModel: wafMock}
	logSvc := NewLogService(context.Background(), svcCtx)
	resp, err := logSvc.GetSiteMetrics(&types.SiteMetricsReq{
		Hosts:         []string{"api.example.com", "web.example.com"},
		WindowMinutes: 30,
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp.List[0].Total != 20 || resp.List[0].Count5xx != 7 || resp.List[0].Count4xx != 0 || resp.List[0].Blocked != 3 {
		t.Fatalf("api metrics: %+v", resp.List[0])
	}
	if resp.List[1].Total != 0 || resp.List[1].Count5xx != 0 || resp.List[1].Count4xx != 0 || resp.List[1].Blocked != 0 {
		t.Fatalf("web should be zero: %+v", resp.List[1])
	}
}
