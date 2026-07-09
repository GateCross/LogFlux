package caddy

import (
	"context"
	"fmt"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DiscoverDockerServicesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	// 测试可注入；nil 时使用默认 Docker 列表实现
	listFn listDockerContainersFn
}

func NewDiscoverDockerServicesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DiscoverDockerServicesLogic {
	return &DiscoverDockerServicesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 扫描 Docker 标签并返回仅会话级候选。
// 硬约束：
//   - 不调用 Caddy /load
//   - 不写发现库
//   - Docker 为空/不可用时返回空列表 + 中文提示（socket 缺失不硬崩溃）
func (l *DiscoverDockerServicesLogic) DiscoverDockerServices() (resp *types.DockerDiscoveryResp, err error) {
	listFn := l.listFn
	if listFn == nil {
		listFn = defaultListDockerContainers
	}

	ctx := l.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// 独立限制发现耗时，避免受父请求取消边界情况影响
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	scannedAt := time.Now().Format("2006-01-02 15:04:05")
	containers, listErr := listFn(ctx)
	if listErr != nil {
		// 软失败：返回空候选 + 中文错误，便于 UI 展示引导
		// Docker 不可用不返回 HTTP 错误——发现为可选 Phase 2 能力
		l.Infof("Docker 发现扫描失败: %v", listErr)
		return &types.DockerDiscoveryResp{
			List:      []types.DockerDiscoveryCandidate{},
			ScannedAt: scannedAt,
			Message:   listErr.Error(),
		}, nil
	}

	parsed := ParseDockerDiscoveryCandidates(containers)
	list := make([]types.DockerDiscoveryCandidate, 0, len(parsed))
	for _, c := range parsed {
		list = append(list, types.DockerDiscoveryCandidate{
			CandidateId:    c.CandidateId,
			ContainerId:    c.ContainerId,
			ContainerName:  c.ContainerName,
			Status:         c.Status,
			Name:           c.Name,
			Domains:        ensureStringSlice(c.Domains),
			Upstream:       c.Upstream,
			LbPolicy:       c.LbPolicy,
			TlsMode:        c.TlsMode,
			HealthPath:     c.HealthPath,
			HealthInterval: c.HealthInterval,
			HealthTimeout:  c.HealthTimeout,
			Reason:         c.Reason,
			Valid:          c.Valid,
		})
	}

	msg := fmt.Sprintf("已扫描 %d 个运行中容器，匹配 %d 个候选（仅会话草稿，未写入数据库，未调用 /load）",
		len(containers), len(list))

	return &types.DockerDiscoveryResp{
		List:      list,
		ScannedAt: scannedAt,
		Message:   msg,
	}, nil
}

func ensureStringSlice(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
