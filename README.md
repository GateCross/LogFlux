# LogFlux

日志流量分析与安全升级管理系统（Caddy + Go + Vue）。

## 快速开始

### Docker 部署（推荐）

1. 克隆项目

```bash
git clone https://github.com/GateCross/LogFlux.git
cd LogFlux
```

2. 准备部署文件

```bash
cp docker/.env.example docker/.env
cp docker/config.example.yaml backend/etc/config.yaml
```

3. 修改配置（重点）

- `backend/etc/config.yaml`
  - `Auth.AccessSecret`（生产环境务必修改）
  - `Database.*`
  - `Redis.*`（可选）
  - `Waf.CorazaReleaseAPI`（默认可用）
  - `Waf.CorazaCheckProxy`（访问 GitHub 受限时配置）

4. 启动服务

```bash
docker compose -f docker/docker-compose.yml up -d
# 或使用 Makefile
make up
```

5. 验证

```bash
docker compose -f docker/docker-compose.yml ps
curl -f http://localhost/api/health
```

默认访问：

- HTTP: `http://localhost`
- HTTPS: `https://localhost`

> 使用预构建镜像时，可在 `docker/.env` 设置：
>
> ```env
> LOGFLUX_IMAGE=ghcr.io/<owner>/<repo>:latest
> ```

完整部署说明见：[`docker/README.md`](docker/README.md)

## 关键能力

- Caddy 反向代理与日志采集
- 日志分析与归档
- RBAC 权限控制
- 安全升级管理（CRS）
- Coraza 引擎版本检查（基于 GitHub Release）

## 架构概览

```text
Client -> Caddy(80/443) -> Frontend + Backend(8888)
                          -> PostgreSQL / Redis(可选)
```

## 本地开发

### 前端

```bash
cd frontend
pnpm install
pnpm run dev
```

### 后端

```bash
cd backend
go mod download
go run logflux.go -f etc/config.yaml
```

## 常用运维命令

```bash
make status
make logs
make restart
make down
```

## 许可证

MIT
