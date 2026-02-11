# LogFlux 安全管理前端可配置能力扩展整体方案（Coraza / CRS）

> 更新时间：2026-02-11
> 适用范围：LogFlux `安全管理` 页面（WAF/CRS）后续迭代

## 1. 背景与目标

当前系统已具备 **WAF 更新管理** 的完整链路（更新源、同步、上传、激活、回滚、任务日志、Coraza 版本检查），但仍缺少“运行时安全策略可视化配置”能力。

本方案目标：

1. 在不破坏现有更新链路的前提下，补齐 Coraza/CRS 的关键运行时配置能力。
2. 将“写死在配置文件中的高价值安全参数”逐步转为前端可控（并保留回滚能力）。
3. 参考 Coraza 生态常见实现（Coraza-Caddy / APISIX / Higress 的插件化配置思路），形成分阶段可落地路线。

---

## 2. 当前能力现状（基于仓库代码）

## 2.1 已有能力

- 更新源管理：`remote/manual`、代理、鉴权、调度、自动检查/下载/激活。
- 规则包上传：支持 `.zip/.tar.gz`，支持校验和立即激活。
- 版本管理：发布列表、手动激活、回滚、清理非激活版本。
- 审计能力：任务日志（check/download/verify/activate/rollback/engine_check）。
- 引擎检查：Coraza 上游版本检查（不在线升级二进制）。

## 2.2 当前边界

- Coraza 引擎不支持在线激活/同步（仅版本检查）。
- 安全运行参数仍主要固定在 `docker/Caddyfile`：
  - `SecRuleEngine On`
  - `SecAuditEngine RelevantOnly`
  - `SecRequestBodyLimit` / `SecRequestBodyNoFilesLimit`
- 系统配置层（`Waf.AllowedDomains`、大小限制、解压限制等）尚未前端可配。
- `schedule/autoCheck/autoDownload` 已建模，但 WAF 专属调度器尚未完全落地到执行层。

## 2.3 核心问题

1. 安全策略与规则版本管理割裂：只能换“包”，不能调“策略”。
2. 缺少低风险调优路径：如 `DetectionOnly` 观察模式、PL/阈值渐进调优。
3. 缺少结构化“例外管理”：业务误报只能靠手改配置。

---

## 3. 对标 Coraza 开源实践的可配置项清单

建议新增以下前端可配置能力（按优先级排序）：

## 3.1 引擎运行模式

- `SecRuleEngine`: `On | Off | DetectionOnly`
- 目标：支持“先检测后阻断”的灰度策略。

## 3.2 审计日志策略

- `SecAuditEngine`
- `SecAuditLogFormat`
- `SecAuditLogParts`
- `SecAuditLogRelevantStatus`
- 目标：在性能与可观测性间可控平衡。

## 3.3 请求/响应体安全限制

- `SecRequestBodyAccess`
- `SecRequestBodyLimit`
- `SecRequestBodyNoFilesLimit`
- （可选）`SecResponseBodyAccess` 与响应体限制
- 目标：防止大包绕过/资源耗尽类风险。

## 3.4 CRS 调优参数

- `tx.paranoia_level`
- `tx.inbound_anomaly_score_threshold`
- `tx.outbound_anomaly_score_threshold`
- 目标：提供“低误报 / 平衡 / 高拦截”预设模板。

## 3.5 规则例外（误报治理）

- `SecRuleRemoveById` / `SecRuleRemoveByTag`
- 按路径/主机/方法的 `ctl:ruleRemoveById/Tag` 范围豁免
- 目标：把临时手工改配置，沉淀为可追踪策略对象。

## 3.6 策略作用域与优先级

- 全局策略
- 站点（host）策略
- 路由（path）策略
- 优先级建议：`全局 < 站点 < 路由`

---

## 4. 目标架构（前后端）

## 4.1 数据模型建议

1. `waf_policies`
   - `name`、`mode`、`audit_config`、`body_limit_config`、`crs_tuning_config`、`enabled`、`version`
2. `waf_policy_bindings`
   - `policy_id`、`scope_type(global/site/route)`、`scope_value`、`priority`
3. `waf_rule_exclusions`
   - `policy_id`、`matchers(host/path/method)`、`remove_type(id/tag)`、`remove_value`
4. `waf_policy_revisions`
   - 发布快照、回滚、审计信息

## 4.2 配置生成与生效

1. 前端提交策略对象。
2. 后端生成 Coraza 指令片段（结构化 -> directives）。
3. 写入策略快照并与当前 CRS 版本组合。
4. 执行 `/adapt` 预校验 + `/load` 热加载。
5. 失败自动回滚到上一策略快照与 `last_good` 版本。

---

## 5. 前端页面信息架构建议

在现有 `安全管理` 页面新增标签页：

1. `更新管理`（保留现有 source/release/job）
2. `运行模式`（SecRuleEngine、基础审计）
3. `CRS 调优`（PL + anomaly 阈值 + 预设模板）
4. `规则例外`（按 ID/Tag 的豁免列表）
5. `策略绑定`（全局/站点/路由）
6. `发布与回滚`（策略版本历史）

每个页签统一支持：

- 变更预览（生成的 directives 差异）
- dry-run 校验（仅 adapt）
- 正式发布（adapt + load）
- 一键回滚

---

## 6. API 草案（建议）

前缀：`/api/caddy/waf`

1. `GET /policy`：策略列表
2. `POST /policy`：创建策略
3. `PUT /policy/:id`：更新策略
4. `DELETE /policy/:id`：删除策略
5. `POST /policy/:id/preview`：生成配置预览
6. `POST /policy/:id/validate`：仅适配校验
7. `POST /policy/:id/publish`：发布生效
8. `POST /policy/rollback`：按 revision 回滚
9. `GET /policy/revision`：策略发布历史
10. `GET /policy/exclusion` / `POST /policy/exclusion`：例外规则管理
11. `GET /policy/binding` / `POST /policy/binding`：策略绑定管理

---

## 7. 分阶段开发计划

## P0（1 周）：运行时基础可配

目标：先把最高价值、最低风险能力开放。

交付：

1. 后端：`waf_policies` + `waf_policy_revisions` 基础模型与 API。
2. 前端：新增 `运行模式` 页签（引擎模式 + 审计 + 请求体限制）。
3. 发布链路：`preview/validate/publish/rollback` 基础打通。

验收：

- 可在 UI 中切换 `On/DetectionOnly`。
- 发布失败可自动回滚。

## P1（1 周）：CRS 调优模板化

交付：

1. 前端：`CRS 调优` 页签。
2. 预设模板：低误报/平衡/高拦截。
3. 发布前风险提示（例如 PL>=3 提示）。

验收：

- 可独立发布 CRS 调优参数。
- 模板切换可追溯到 revision。

## P2（1~2 周）：误报治理与作用域

交付：

1. 前端：`规则例外` + `策略绑定` 页签。
2. 后端：按 host/path/method 生成局部豁免配置。
3. 优先级冲突检测与告警。

验收：

- 支持全局/站点/路由策略叠加。
- 同一路由冲突可被阻止发布。

## P3（1 周）：可观测与运营化

交付：

1. 增加策略命中统计面板（拦截/放行/误报反馈）。
2. 变更审计增强（谁在何时改了什么）。
3. 与通知系统联动（发布失败、回滚事件）。

验收：

- 运维可通过面板判断策略效果并指导调优。

---

## 8. 风险与应对

1. **误操作导致全站拦截**
   - 应对：默认 `DetectionOnly` 灰度 + 发布前 dry-run + 快速回滚按钮。
2. **配置复杂度升高**
   - 应对：模板优先、专家模式后置、字段分层展示。
3. **性能波动**
   - 应对：高 PL 提示 + 审计级别建议 + 运行指标看板。
4. **策略与规则版本不一致**
   - 应对：策略 revision 绑定 CRS release，统一发布记录。

---

## 9. DoD（完成定义）

满足以下条件视为“整体方案落地完成”：

1. 运行时核心配置可在前端完成闭环（预览/校验/发布/回滚）。
2. 支持 CRS 调优与规则例外管理，且具备审计记录。
3. 策略可按全局/站点/路由生效，具备冲突检测。
4. 发布失败自动回滚稳定可用。
5. 与现有规则更新管理（source/release/job）可并行协作，无破坏。

