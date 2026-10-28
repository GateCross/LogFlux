# LogFlux 文档

本目录只保留与当前代码一致的文档。历史设计与实施计划已合并归档，不再单独维护。

## 文档清单

| 文档 | 内容 |
|---|---|
| [`caddy-architecture.md`](./caddy-architecture.md) | Caddy 配置架构：单一真相源、单实例模型、站点能力、证书中心、站点日志、未落地技术债 |
| [`waf.md`](./waf.md) | WAF（Coraza/CRS）：运行时策略、规则仓库布局、更新接口与流程、通知事件、运维处置 |
| [`telegram-setup-guide.md`](./telegram-setup-guide.md) | Telegram 通知渠道配置 |

## 其他入口

- 项目总览与快速开始：[`../README.md`](../README.md)
- Docker 部署：[`../docker/README.md`](../docker/README.md)
- 开发约定与分层规范：[`../contexts/context.md`](../contexts/context.md)

## 维护约定

1. **文档描述当前代码，不描述计划。** 未落地的能力要么不写，要么明确标注「未落地」并说明目标，不要写成已完成。
2. 部署流程、环境变量、挂载目录变化时，优先更新 `docker/README.md`、`docker/config.example.yaml`、根 `README.md`。
3. 设计讨论、方案对比、任务清单**不进仓库**——结论直接改进上面三篇文档，过程留在提交信息或 issue 里。
4. 删除页面/接口时，同步删除对应文档段落。曾因 `/security` 页面整体删除后文档未清理，导致多人按不存在的能力设计实现。
5. 文档与实现冲突时，以代码为准，并立即修正文档。

---

最后更新：2026-10-28
