package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LogFlux Docker 发现标签（仅会话候选；无发现库）。
//
//	logflux.enable=true|1|yes          必填
//	logflux.host=app.example.com       必填；逗号分隔域名
//	logflux.port=8080                  容器私有端口（默认首个暴露端口 / 80）
//	logflux.name=My App                可选站点名称
//	logflux.tls=auto|off|internal      可选（默认 auto）
//	logflux.lb_policy=round_robin|...  可选
//	logflux.upstream=host:port         可选覆盖上游
//	logflux.health.path=/health        可选
//	logflux.health.interval=10s        可选
//	logflux.health.timeout=5s          可选
const (
	labelEnable         = "logflux.enable"
	labelHost           = "logflux.host"
	labelPort           = "logflux.port"
	labelName           = "logflux.name"
	labelTLS            = "logflux.tls"
	labelLBPolicy       = "logflux.lb_policy"
	labelUpstream       = "logflux.upstream"
	labelHealthPath     = "logflux.health.path"
	labelHealthInterval = "logflux.health.interval"
	labelHealthTimeout  = "logflux.health.timeout"
)

// 标签发现使用的最小容器视图。
// 纯数据：不依赖 Docker SDK；由 Engine API 或测试填充。
type DockerContainerSummary struct {
	ID     string
	Name   string
	Status string
	Labels map[string]string
	// 容器侧 TCP 端口（来自 Ports[]）。
	PrivatePorts []int
	// 网络名 → IP 映射（部分 list 响应可能为空）。
	NetworkIPs map[string]string
}

// 由标签推导的会话级草稿建议。
// 不落库到并行发现库；Apply_Path 仍需用户确认。
type DockerDiscoveryCandidate struct {
	CandidateId    string            `json:"candidateId"`
	ContainerId    string            `json:"containerId"`
	ContainerName  string            `json:"containerName"`
	Status         string            `json:"status"`
	Name           string            `json:"name"`
	Domains        []string          `json:"domains"`
	Upstream       string            `json:"upstream"`
	LbPolicy       string            `json:"lbPolicy,optional"`
	TlsMode        string            `json:"tlsMode"`
	HealthPath     string            `json:"healthPath,optional"`
	HealthInterval string            `json:"healthInterval,optional"`
	HealthTimeout  string            `json:"healthTimeout,optional"`
	Labels         map[string]string `json:"labels,optional"`
	Reason         string            `json:"reason,optional"` // 无效时的跳过原因
	Valid          bool              `json:"valid"`
}

// 列出用于发现的运行中容器（测试可注入）。
type listDockerContainersFn func(ctx context.Context) ([]DockerContainerSummary, error)

// 将容器标签转为会话候选。
// 无 logflux.enable 的容器忽略；enable 但标签无效时 Valid=false 且 Reason 为中文。
// 不触碰 Caddy /load 与任何数据库。
func ParseDockerDiscoveryCandidates(containers []DockerContainerSummary) []DockerDiscoveryCandidate {
	out := make([]DockerDiscoveryCandidate, 0)
	for _, c := range containers {
		labels := c.Labels
		if labels == nil {
			continue
		}
		if !isTruthyLabel(labels[labelEnable]) {
			continue
		}

		name := strings.TrimSpace(firstNonEmptyLabel(labels[labelName], stripContainerSlash(c.Name), shortContainerID(c.ID)))
		domains := splitCSV(labels[labelHost])
		tlsMode := normalizeTLSMode(labels[labelTLS])
		lbPolicy := normalizeLBPolicy(labels[labelLBPolicy])
		upstream := strings.TrimSpace(labels[labelUpstream])
		if upstream == "" {
			upstream = deriveUpstream(c, labels[labelPort])
		}

		healthPath := strings.TrimSpace(labels[labelHealthPath])
		healthInterval := strings.TrimSpace(labels[labelHealthInterval])
		healthTimeout := strings.TrimSpace(labels[labelHealthTimeout])

		cand := DockerDiscoveryCandidate{
			CandidateId:    buildCandidateID(c.ID, domains, upstream),
			ContainerId:    c.ID,
			ContainerName:  stripContainerSlash(c.Name),
			Status:         c.Status,
			Name:           name,
			Domains:        domains,
			Upstream:       upstream,
			LbPolicy:       lbPolicy,
			TlsMode:        tlsMode,
			HealthPath:     healthPath,
			HealthInterval: healthInterval,
			HealthTimeout:  healthTimeout,
			Labels:         copyStringMap(labels),
			Valid:          true,
		}

		if len(domains) == 0 {
			cand.Valid = false
			cand.Reason = "缺少 logflux.host 标签（域名）"
		} else if strings.TrimSpace(upstream) == "" {
			cand.Valid = false
			cand.Reason = "无法推断上游地址：请设置 logflux.upstream 或 logflux.port"
		} else if healthPath != "" && !strings.HasPrefix(healthPath, "/") {
			cand.Valid = false
			cand.Reason = "健康检查路径必须以 / 开头"
		}

		out = append(out, cand)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Valid != out[j].Valid {
			return out[i].Valid
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func deriveUpstream(c DockerContainerSummary, portLabel string) string {
	port := strings.TrimSpace(portLabel)
	if port == "" && len(c.PrivatePorts) > 0 {
		// 优先使用第一个私有端口
		sort.Ints(c.PrivatePorts)
		port = strconv.Itoa(c.PrivatePorts[0])
	}
	if port == "" {
		port = "80"
	}

	// 共享 Docker 网络时优先用容器名（LogFlux 典型部署）
	cname := stripContainerSlash(c.Name)
	if cname != "" {
		return fmt.Sprintf("%s:%s", cname, port)
	}
	// 回退到第一个网络 IP
	if len(c.NetworkIPs) > 0 {
		names := make([]string, 0, len(c.NetworkIPs))
		for n := range c.NetworkIPs {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			ip := strings.TrimSpace(c.NetworkIPs[n])
			if ip != "" {
				return fmt.Sprintf("%s:%s", ip, port)
			}
		}
	}
	return ""
}

func buildCandidateID(containerID string, domains []string, upstream string) string {
	hostPart := "nohost"
	if len(domains) > 0 {
		hostPart = domains[0]
	}
	short := shortContainerID(containerID)
	if short == "" {
		short = "unknown"
	}
	return fmt.Sprintf("docker-%s-%s-%s", short, sanitizeIDPart(hostPart), sanitizeIDPart(upstream))
}

func sanitizeIDPart(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "x"
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := b.String()
	if len(out) > 40 {
		return out[:40]
	}
	return out
}

func shortContainerID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func stripContainerSlash(name string) string {
	name = strings.TrimSpace(name)
	return strings.TrimPrefix(name, "/")
}

func isTruthyLabel(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on", "enable", "enabled":
		return true
	default:
		return false
	}
}

func splitCSV(v string) []string {
	parts := strings.FieldsFunc(v, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

func normalizeTLSMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "off", "false", "0", "http":
		return "off"
	case "internal":
		return "internal"
	default:
		return "auto"
	}
}

func normalizeLBPolicy(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "least_conn", "least-conn", "leastconn":
		return "least_conn"
	case "ip_hash", "ip-hash", "iphash":
		return "ip_hash"
	case "round_robin", "round-robin", "roundrobin", "":
		if strings.TrimSpace(v) == "" {
			return "round_robin"
		}
		return "round_robin"
	default:
		// 未知策略统一回落 round_robin；发现仅保留简单策略
		if v == "" {
			return "round_robin"
		}
		return "round_robin"
	}
}

func firstNonEmptyLabel(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// ---------- Docker Engine API 客户端（无 SDK；仅 unix/tcp） ----------

type dockerListItem struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Labels map[string]string `json:"Labels"`
	Status string            `json:"Status"`
	State  string            `json:"State"`
	Ports  []struct {
		PrivatePort int    `json:"PrivatePort"`
		PublicPort  int    `json:"PublicPort"`
		Type        string `json:"Type"`
		IP          string `json:"IP"`
	} `json:"Ports"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string   `json:"IPAddress"`
			Aliases   []string `json:"Aliases"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

// 调用 Docker Engine API 列出运行中容器（只读）。
// 不调用 Caddy /load。错误信息为中文，供 UI 展示。
func defaultListDockerContainers(ctx context.Context) ([]DockerContainerSummary, error) {
	client, baseURL, err := newDockerHTTPClient()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/containers/json?all=false", nil)
	if err != nil {
		return nil, fmt.Errorf("构造 Docker 请求失败: %w", err)
	}
	// Docker API 版本协商头可选；多数安装可接受无版本路径
	req.Header.Set("User-Agent", "LogFlux-DockerDiscovery/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接 Docker 失败: %w（请挂载 /var/run/docker.sock 或设置 DOCKER_HOST）", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("读取 Docker 响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("Docker API 返回错误: %s", truncateDiscoveryMessage(msg))
	}

	var items []dockerListItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("解析 Docker 容器列表失败: %w", err)
	}

	out := make([]DockerContainerSummary, 0, len(items))
	for _, it := range items {
		name := ""
		if len(it.Names) > 0 {
			name = it.Names[0]
		}
		ports := make([]int, 0)
		seenPort := map[int]struct{}{}
		for _, p := range it.Ports {
			if strings.EqualFold(p.Type, "tcp") || p.Type == "" {
				if p.PrivatePort > 0 {
					if _, ok := seenPort[p.PrivatePort]; !ok {
						seenPort[p.PrivatePort] = struct{}{}
						ports = append(ports, p.PrivatePort)
					}
				}
			}
		}
		ips := map[string]string{}
		for netName, netInfo := range it.NetworkSettings.Networks {
			if strings.TrimSpace(netInfo.IPAddress) != "" {
				ips[netName] = netInfo.IPAddress
			}
		}
		status := strings.TrimSpace(it.Status)
		if status == "" {
			status = strings.TrimSpace(it.State)
		}
		out = append(out, DockerContainerSummary{
			ID:           it.ID,
			Name:         name,
			Status:       status,
			Labels:       it.Labels,
			PrivatePorts: ports,
			NetworkIPs:   ips,
		})
	}
	return out, nil
}

func newDockerHTTPClient() (*http.Client, string, error) {
	host := strings.TrimSpace(os.Getenv("DOCKER_HOST"))
	if host == "" {
		if runtime.GOOS == "windows" {
			// 优先 TCP（若用户已暴露）；命名管道需额外依赖
			host = "npipe:////./pipe/docker_engine"
		} else {
			host = "unix:///var/run/docker.sock"
		}
	}

	switch {
	case strings.HasPrefix(host, "unix://"):
		sock := strings.TrimPrefix(host, "unix://")
		if _, err := os.Stat(sock); err != nil {
			return nil, "", fmt.Errorf("Docker socket 不可用: %s", sock)
		}
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sock)
			},
		}
		return &http.Client{Transport: transport, Timeout: 8 * time.Second}, "http://docker", nil

	case strings.HasPrefix(host, "tcp://"):
		base := "http://" + strings.TrimPrefix(host, "tcp://")
		return &http.Client{Timeout: 8 * time.Second}, strings.TrimRight(base, "/"), nil

	case strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://"):
		return &http.Client{Timeout: 8 * time.Second}, strings.TrimRight(host, "/"), nil

	case strings.HasPrefix(host, "npipe:"):
		return nil, "", fmt.Errorf("当前环境不支持命名管道 Docker 连接，请设置 DOCKER_HOST=tcp://localhost:2375 或在 Linux 挂载 docker.sock")

	default:
		return nil, "", fmt.Errorf("不支持的 DOCKER_HOST: %s", host)
	}
}

func truncateDiscoveryMessage(msg string) string {
	const max = 240
	runes := []rune(msg)
	if len(runes) <= max {
		return msg
	}
	return string(runes[:max]) + "..."
}
