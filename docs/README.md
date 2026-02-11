# LogFlux 文档中心

本文档按当前仓库代码与部署方式整理，优先覆盖 Docker 部署与安全管理（CRS/Coraza）。

## 快速入口

- 项目总览与快速开始：[`README.md`](../README.md)
- Docker 部署主文档：[`docker/README.md`](../docker/README.md)
- Docker 配置模板：[`docker/config.example.yaml`](../docker/config.example.yaml)
- Compose 编排文件：[`docker/docker-compose.yml`](../docker/docker-compose.yml)

## 部署文档（当前有效）

### 1) Docker 一体化部署（推荐）

请直接参考：[`docker/README.md`](../docker/README.md)

覆盖内容包括：

- 环境准备与配置项说明
- 镜像使用与本地构建
- Coraza 版本检查机制（GitHub Release）
- 代理配置与失败回退策略
- 常见故障排查与运维命令

### 2) 关键配置文件

- 后端运行配置模板：[`docker/config.example.yaml`](../docker/config.example.yaml)
- 容器编排：[`docker/docker-compose.yml`](../docker/docker-compose.yml)
- 本地环境变量模板：[`docker/.env.example`](../docker/.env.example)

## 安全管理相关文档

以下文档主要用于设计背景、实施计划和运维规范，部分内容可能早于当前实现，请以代码与 `docker/README.md` 的“当前行为”章节为准：

- [`plans/caddy-coraza-crs-full-cutover-plan.md`](./plans/caddy-coraza-crs-full-cutover-plan.md)
- [`plans/waf-update-management-design.md`](./plans/waf-update-management-design.md)
- [`plans/waf-update-p0-implementation-plan.md`](./plans/waf-update-p0-implementation-plan.md)
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

最后更新：2026-02-11
