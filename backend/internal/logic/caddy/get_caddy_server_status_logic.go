package caddy

import (
	"context"
	"strings"
	"sync"
	"time"

	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/safego"
	"logflux/internal/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

// 并行 Admin API 探测上限，多节点列表保持响应且不无限开连接。
// 设计：信号量约 4–8。
const probeConcurrency = 6

// 状态发现使用的只读探测函数。
// 生产走 GET {url}/config/；测试可注入 mock。
type probeCaddyServerFn func(server *caddymodel.CaddyServer) error

type GetCaddyServerStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	// 测试可注入；nil 时使用默认配置拉取
	probeFn probeCaddyServerFn
}

func NewGetCaddyServerStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyServerStatusLogic {
	return &GetCaddyServerStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyServerStatusLogic) GetCaddyServerStatus() (resp *types.CaddyServerStatusListResp, err error) {
	var servers []caddymodel.CaddyServer
	if err := l.svcCtx.DB.WithContext(l.ctx).Find(&servers).Error; err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询 Caddy 节点失败", err)
	}

	probe := l.probeFn
	if probe == nil {
		// 默认探测透传请求 context，便于父请求取消时中止 in-flight GET /config/
		probe = func(server *caddymodel.CaddyServer) error {
			_, err := getCaddyConfigJSON(l.ctx, server)
			return err
		}
	}

	list := probeCaddyServers(l.ctx, servers, probe)
	return &types.CaddyServerStatusListResp{List: list}, nil
}

// 对每个已注册节点做有界并发探测。
// 部分失败不丢弃成功结果；空输入返回非 nil 空切片。
// 探测路径只读（GET /config/），禁止调用 /load。
// 探测协程经 safego 启动，panic 不会打崩进程。
func probeCaddyServers(ctx context.Context, servers []caddymodel.CaddyServer, probe probeCaddyServerFn) []types.CaddyServerStatusItem {
	if len(servers) == 0 {
		return []types.CaddyServerStatusItem{}
	}
	if probe == nil {
		probe = func(server *caddymodel.CaddyServer) error {
			_, err := getCaddyConfigJSON(ctx, server)
			return err
		}
	}

	results := make([]types.CaddyServerStatusItem, len(servers))
	sem := make(chan struct{}, probeConcurrency)
	var wg sync.WaitGroup

	for i := range servers {
		wg.Add(1)
		idx := i
		// 使用 safego 包装，避免探测 panic 打崩进程；外层不再裸 go
		safego.New(ctx, "Caddy 节点状态探测").Go(func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			server := servers[idx]
			results[idx] = probeOneServer(&server, probe)
		})
	}
	wg.Wait()
	return results
}

func probeOneServer(server *caddymodel.CaddyServer, probe probeCaddyServerFn) types.CaddyServerStatusItem {
	probedAt := time.Now().Format("2006-01-02 15:04:05")
	item := types.CaddyServerStatusItem{
		ServerId:  server.ID,
		Name:      server.Name,
		Reachable: false,
		LatencyMs: 0,
		ProbedAt:  probedAt,
	}

	start := time.Now()
	err := probe(server)
	latency := time.Since(start).Milliseconds()
	if latency < 0 {
		latency = 0
	}
	item.LatencyMs = latency
	item.ProbedAt = time.Now().Format("2006-01-02 15:04:05")

	if err != nil {
		item.ErrorMessage = summarizeProbeError(err)
		return item
	}
	item.Reachable = true
	return item
}

// 返回面向 UI 的简短中文摘要；技术细节会截断。
func summarizeProbeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "探测失败"
	}
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return "探测超时：无法在限定时间内连接 Caddy Admin API"
	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "connectex"):
		return "连接被拒绝：Caddy Admin API 不可达"
	case strings.Contains(lower, "no such host") || strings.Contains(lower, "lookup"):
		return "主机解析失败：无法解析 Caddy 节点地址"
	case strings.Contains(lower, "unauthorized") || strings.Contains(msg, "401"):
		return "鉴权失败：Token 无效或未授权"
	case strings.Contains(msg, "403"):
		return "鉴权失败：禁止访问 Caddy Admin API"
	case strings.HasPrefix(msg, "获取 Caddy 配置失败"):
		// 已是配置拉取返回的中文错误
		return truncateProbeMessage(msg)
	default:
		return truncateProbeMessage("探测失败: " + msg)
	}
}

func truncateProbeMessage(msg string) string {
	const max = 200
	if len(msg) <= max {
		return msg
	}
	// 按 rune 截断，兼容中英文混合
	runes := []rune(msg)
	if len(runes) <= max {
		return msg
	}
	return string(runes[:max]) + "..."
}
