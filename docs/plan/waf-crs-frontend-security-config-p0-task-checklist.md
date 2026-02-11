# LogFlux 安全管理前端可配置能力扩展（P0 执行清单）

> 基于文档：`docs/plan/waf-crs-frontend-security-config-overall-plan.md`
> 目标周期：1 周（建议）
> 优先级：P0

## 1. P0 范围定义（冻结）

本阶段只做“运行时基础可配”，不做作用域和复杂例外。

## 1.1 In Scope

1. 引擎模式：`SecRuleEngine`（`On/Off/DetectionOnly`）
2. 审计基础：`SecAuditEngine`、`SecAuditLogFormat`、`SecAuditLogRelevantStatus`
3. 请求体控制：`SecRequestBodyAccess`、`SecRequestBodyLimit`、`SecRequestBodyNoFilesLimit`
4. 配置预览与发布：`preview/validate/publish/rollback`
5. 发布历史：最小化 revision 记录

## 1.2 Out of Scope

1. 规则例外管理（`removeById/removeByTag/ctl`）
2. 作用域绑定（全局/站点/路由）
3. 调度器能力改造
4. 可观测面板与高级报表

---

## 2. 任务分解（可执行）

## A. 数据模型与迁移（后端）

- [x] `P0-A01` 新增模型：`backend/model/waf_policy.go`
- [x] `P0-A02` 新增模型：`backend/model/waf_policy_revision.go`
- [x] `P0-A03` 在 `backend/internal/svc/service_context.go` 加入 AutoMigrate
- [x] `P0-A04` 增加默认全局策略初始化（如 `default-global-policy`）
- [x] `P0-A05` 为策略配置定义结构化 JSON 字段（避免纯文本拼接）

建议字段（最小集）：

- `waf_policies`
  - `id/name/enabled/is_default`
  - `engine_mode`（on/off/detectiononly）
  - `audit_engine`（off/on/relevantonly）
  - `audit_log_format`（json/native）
  - `audit_relevant_status`（字符串）
  - `request_body_access`（bool）
  - `request_body_limit`（int）
  - `request_body_no_files_limit`（int）
  - `created_at/updated_at`
- `waf_policy_revisions`
  - `id/policy_id/version/status`（draft/published/rolled_back）
  - `config_snapshot`（jsonb）
  - `directives_snapshot`（text）
  - `operator/message/created_at`

## B. API 设计与生成（go-zero）

- [x] `P0-B01` 在 `backend/api/manage.api` 新增策略类型：
  - `WafPolicy*Req/WafPolicy*Resp`
- [x] `P0-B02` 新增接口：
  - `GET /caddy/waf/policy`
  - `POST /caddy/waf/policy`
  - `PUT /caddy/waf/policy/:id`
  - `DELETE /caddy/waf/policy/:id`
  - `POST /caddy/waf/policy/:id/preview`
  - `POST /caddy/waf/policy/:id/validate`
  - `POST /caddy/waf/policy/:id/publish`
  - `POST /caddy/waf/policy/rollback`
  - `GET /caddy/waf/policy/revision`
- [ ] `P0-B03` 执行 goctl 代码生成并校验路由接入
- [x] `P0-B04` 在 `backend/internal/types/types.go` 核对字段 tag 与默认值

## C. 后端逻辑实现（caddy logic）

- [x] `P0-C01` 新建策略构建器：`backend/internal/logic/caddy/waf_policy_builder.go`
- [x] `P0-C02` 实现结构化配置 -> Coraza directives 的确定性渲染
- [x] `P0-C03` 实现 `preview`：返回渲染结果（不落地）
- [x] `P0-C04` 实现 `validate`：调用 `/adapt` 做 dry-run
- [x] `P0-C05` 实现 `publish`：
  - 落 revision
  - 调用 `/adapt` + `/load`
  - 成功标记 published
- [x] `P0-C06` 实现 `rollback`：按 revision 回滚并重载
- [ ] `P0-C07` 发布失败时与现有回滚链路统一（复用 `last_good` 思路）
- [ ] `P0-C08` 统一错误信息本地化（与现有 WAF job message 风格一致）

实现约束：

1. 禁止前端直接提交 directives 原文，必须提交结构化字段。
2. 所有整数阈值做上下限校验（避免误填超大值）。
3. 对 `DetectionOnly` 给予风险提示信息（响应字段返回 warning）。

## D. 前端实现（安全管理页）

- [ ] `P0-D01` 在 `frontend/src/service/api/caddy.ts` 增加策略 API 封装
- [ ] `P0-D02` 在 `frontend/src/views/security/index.vue` 新增 `运行模式` Tab
- [ ] `P0-D03` 新增表单字段：
  - engine mode
  - audit engine/log format/relevant status
  - request body access/limits
- [ ] `P0-D04` 新增 `预览` 按钮（展示 directives 结果）
- [ ] `P0-D05` 新增 `校验` 按钮（dry-run）
- [ ] `P0-D06` 新增 `发布`、`回滚` 交互
- [ ] `P0-D07` 新增“最近发布记录”表格（最小字段）
- [ ] `P0-D08` 表单校验：
  - 数值范围
  - 必填
  - `relevant_status` 正则格式

交互建议：

1. 默认首选 `DetectionOnly`（新增策略时）。
2. `On` 模式发布弹二次确认。
3. 发布成功后自动刷新现有 `任务日志` 页签数据。

## E. 测试与验收（后端+前端）

- [ ] `P0-E01` 后端单测：builder 渲染稳定性（同输入同输出）
- [ ] `P0-E02` 后端单测：字段越界/非法枚举校验
- [ ] `P0-E03` 后端单测：publish 失败时回滚分支
- [ ] `P0-E04` API 联调：preview/validate/publish/rollback 全链路
- [ ] `P0-E05` 前端自测：表单校验、发布确认、错误提示
- [ ] `P0-E06` 回归：不影响既有 source/release/job 功能

## F. 文档与运维

- [ ] `P0-F01` 更新 `docs/README.md` 增加策略管理入口说明
- [ ] `P0-F02` 更新 `docker/README.md` 补充策略发布/回滚操作
- [ ] `P0-F03` 补充 FAQ：发布失败常见原因与处理步骤

---

## 3. 联调顺序（建议）

1. 先打通 `preview`（纯后端可验证）。
2. 再接 `validate`（只做 adapt）。
3. 最后接 `publish/rollback`（涉及 load 与回滚）。

这样可把高风险动作放在最后，减少联调成本。

---

## 4. 验收标准（P0 DoD）

- [ ] UI 可配置并保存上述 8 类基础字段。
- [ ] 能看到渲染后的 directives 预览。
- [ ] validate 失败可返回明确错误并阻断发布。
- [ ] publish 成功后策略立即生效。
- [ ] publish 失败自动回滚，业务不中断。
- [ ] 具备 revision 历史，并可手动回滚。
- [ ] 原有 WAF 更新管理功能回归通过。

---

## 5. 里程碑（1 周建议）

- Day 1-2：A+B（模型+API）
- Day 3-4：C（builder+preview+validate+publish）
- Day 5：D（前端页面+联调）
- Day 6：E（测试+回归）
- Day 7：F（文档+验收）

---

## 6. 风险门禁（上线前必过）

1. `publish` 必须先 `validate` 成功。
2. `On` 模式必须二次确认。
3. 回滚接口必须在 staging 演练通过至少 3 次。
4. 生产首发建议默认 `DetectionOnly` 观察 24 小时。
