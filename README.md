# LogFlux

LogFlux 是一个日志流量分析、Caddy 图形化配置与 Coraza WAF 管理系统。

它基于 **go-zero + GORM + PostgreSQL** 构建后端，前端采用 **Vue 3 / Vite / Ant Design Vue / vue-vben-admin**，默认通过 **Caddy** 对外提供统一入口。

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go, go-zero, GORM, PostgreSQL, Redis(可选) |
| 前端 | Vue 3, Vite, TypeScript, Ant Design Vue, vue-vben-admin, Pinia |
| 反代/WAF | Caddy, Coraza, OWASP CRS |
| IP 定位 | ip2region |
| 部署 | Docker Compose, Makefile |

## 核心能力

- **Caddy 配置分块工作台**：可视化编辑简单站点、全局配置、上游池，复杂 Caddyfile 块只读保留不丢失
- Caddy 配置预览、热加载、历史版本与回滚
- Caddy 访问日志、系统日志采集与查询
- **仪表盘数据可视化**：请求趋势、状态码分布、Top 站点、国内/国际地理分布地图
- **IP 区域访问控制**：基于 ip2region 的国家/省份粒度访问限制
- Coraza + OWASP CRS 简单 WAF 管理（检测/阻断/关闭模式、强度调节、审计日志）
- WAF 更新源、发布、绑定、误报反馈与任务审计
- RBAC 权限、用户、角色、菜单管理
- 通知渠道（Telegram 等）、规则、模板、站内通知与日志
- Cron 定时任务管理
- 日志归档与后台调度

## 架构概览

```text
Client -> Caddy(:80) -> Frontend
                     -> Backend(:8888)
                     -> PostgreSQL / Redis(可选)
```

## 快速开始

### Docker 部署（推荐）

1. 准备配置文件

```bash
cp docker/.env.example docker/.env
cp docker/config.example.yaml backend/etc/config.yaml
```

2. 修改后端配置，重点检查

- `Auth.AccessSecret`
- `Database.*`
- `Redis.*`（可选）
- `Waf.CorazaReleaseAPI`
- `Waf.CorazaCheckProxy`（访问 GitHub 受限时配置）

3. 启动服务

```bash
docker compose -f docker/docker-compose.yml up -d
# 或
make up
```

4. 验证

```bash
docker compose -f docker/docker-compose.yml ps
curl -f http://localhost/api/health
```

默认访问地址：

- `http://localhost`

如果使用预构建镜像，可在 `docker/.env` 中设置：

```env
LOGFLUX_IMAGE=ghcr.io/<owner>/<repo>:latest
```

完整部署说明见 [`docker/README.md`](docker/README.md)。

## 本地开发

### 前端

```bash
cd frontend
pnpm install
pnpm run dev:antd
```

### 后端

```bash
cd backend
go mod download
bash scripts/download-xdb.sh
go run logflux.go -f etc/config.yaml
```

`ip2region` 的 `.xdb` 数据文件不提交到仓库；本地首次启动或编译前需要先执行 `bash scripts/download-xdb.sh`，也可以在仓库根目录使用 `make download-xdb`。
`make build-backend`、Docker 与 CI 会在下载后使用 `embed_ipregion` 构建标签把 xdb 嵌入二进制；普通 `go run` / `go build` 会在运行时从 `backend/internal/middleware/data/` 读取。

## 开发约定

- API 定义入口：`backend/api/logflux.api`
- goctl 生成命令：

```bash
cd backend
goctl api go -api api/logflux.api -dir . -style go_zero
```

- 统一响应结构：

```json
{
  "code": 0,
  "message": "成功",
  "msg": "成功",
  "data": {}
}
```

- Handler 使用 `result.HttpResult`
- `internal/types/types.go` 和 `internal/handler/routes.go` 为生成文件，禁止手改
- 业务逻辑优先下沉到 `internal/service/`
- 详细规范见 [`contexts/context.md`](contexts/context.md)

## 前端结构说明

### 当前前端（frontend/）

基于 [vue-vben-admin](https://github.com/vbenjs/vue-vben-admin) (v5) 构建，采用 monorepo 架构（pnpm + Turborepo），Ant Design Vue 组件库。主要业务代码位于 `frontend/apps/src/`：

```
frontend/
  apps/
    src/
      api/           # API 服务层（core/, caddy/, system/ 等）
      router/        # 路由配置 + 动态路由守卫
      store/         # Pinia 状态管理
      views/         # 业务页面
        dashboard/   # 仪表盘
        caddy/       # Caddy 管理（config, access, log, system-log）
        security/    # WAF 管理（source, policy, observe, ops, runtime, crs, exclusion, binding, release, job）
          manage/      # 系统管理（user, role, menu）
          notification/# 通知管理（channel, rule, template, log）
          cron/        # 定时任务
          user/        # 个人中心
        locales/       # 国际化（zh-CN, en-US）
    backend-mock/      # 开发用 Mock API 服务
  packages/            # 共享包（stores, hooks, utils, styles, types 等）
  internal/            # 内部工具（lint-configs, vite-config, tsconfig 等）
```

## 文档入口

- 项目文档索引：[`docs/README.md`](docs/README.md)
- Docker 部署文档：[`docker/README.md`](docker/README.md)
- 项目上下文：[`contexts/context.md`](contexts/context.md)

## 常用运维命令

```bash
make status
make logs
make restart
make down
```

## 许可证

MIT
