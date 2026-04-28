# LogFlux 文档中心

本文档按当前仓库代码与部署方式整理，优先覆盖 Docker 部署、Caddy 图形化配置与简单 WAF 设置。

## 快速入口

- 项目总览与快速开始：[`README.md`](../README.md)
- Docker 部署主文档：[`docker/README.md`](../docker/README.md)
- Docker 配置模板：[`docker/config.example.yaml`](../docker/config.example.yaml)
- Compose 编排文件：[`docker/docker-compose.yml`](../docker/docker-compose.yml)
- WAF/CRS 历史计划与进度：
  - [`plans/waf-crs-frontend-security-config-overall-plan.md`](./plans/waf-crs-frontend-security-config-overall-plan.md)
  - [`plans/waf-crs-frontend-security-config-progress.md`](./plans/waf-crs-frontend-security-config-progress.md)

## 计划文档目录说明

- 当前统一目录为：`docs/plans/`
- 历史 `docs/plan/` 已合并完成，不再维护；如提到“plan”，请直接在 `docs/plans/` 查阅对应文档。

## 部署文档（当前有效）

### 1) Docker 一体化部署（推荐）

请直接参考：[`docker/README.md`](../docker/README.md)

覆盖内容包括：

- 环境准备与配置项说明
- 镜像使用与本地构建
- Caddy 图形化配置、热加载与回滚
- 简单 WAF 设置（关闭 / 仅检测 / 阻断、低误报 / 平衡 / 严格、审计与请求体限制）
- Coraza 版本检查机制（GitHub Release）
- 高级 WAF 能力的保留边界
- 常见故障排查与运维命令

### 2) 关键配置文件

- 后端运行配置模板：[`docker/config.example.yaml`](../docker/config.example.yaml)
- 容器编排：[`docker/docker-compose.yml`](../docker/docker-compose.yml)
- 本地环境变量模板：[`docker/.env.example`](../docker/.env.example)

## 安全管理相关文档

以下文档主要用于设计背景、实施计划和运维规范。当前默认产品入口已收敛到 `Caddy管理 -> Caddy配置 -> 防火墙`；高级安全管理页面默认隐藏，部分历史文档不再代表默认使用路径：

- [`plans/waf-crs-frontend-security-config-overall-plan.md`](./plans/waf-crs-frontend-security-config-overall-plan.md)
- [`plans/waf-crs-frontend-security-config-progress.md`](./plans/waf-crs-frontend-security-config-progress.md)
- [`plans/waf-update-management-design.md`](./plans/waf-update-management-design.md)
- [`plans/waf-update-task-checklist.md`](./plans/waf-update-task-checklist.md)
- [`plans/waf-update-operations-guide.md`](./plans/waf-update-operations-guide.md)

## 其他文档

- Telegram 配置指南：[`telegram-setup-guide.md`](./telegram-setup-guide.md)

## 文档维护约定

- 部署流程、环境变量、挂载目录发生变化时，优先更新：
  - `docker/README.md`
  - `docker/config.example.yaml`
  - `README.md`
- 本文件作为索引，不重复维护长篇操作细节。
- 如设计文档与实现不一致，以运行代码和部署文档为准，并补充更新记录。

---

最后更新：2026-04-28
