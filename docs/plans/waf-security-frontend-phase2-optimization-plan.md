# WAF 安全管理前端第二阶段优化计划（Observe / Release / Job + useWafPolicy）

## 1. 背景与目标

当前 `frontend/src/views/security/index.vue` 仍是超大单文件，模板与逻辑耦合较重。第一阶段已完成 `source` 与 `runtime` 页签的模板拆分，第二阶段目标是继续降低复杂度并为后续重构建立稳定骨架。

本阶段聚焦：

1. 将 `observe`、`release`、`job` 三个页签拆成独立子组件；
2. 启动首个 composable 抽离，落地 `useWafPolicy`；
3. 在**不改变接口协议与业务行为**前提下，提升可维护性与可测试性。

---

## 2. 范围（In Scope）

### 2.1 模板拆分（UI 层）

- 新增子组件：
  - `frontend/src/views/security/tabs/ObserveTabContent.vue`
  - `frontend/src/views/security/tabs/ReleaseTabContent.vue`
  - `frontend/src/views/security/tabs/JobTabContent.vue`
- `index.vue` 保留“页面编排层”职责：
  - tab 容器与路由状态同步；
  - 弹窗挂载与跨域共享状态协调；
  - 数据通过 props/callback 透传到子组件。

### 2.2 逻辑抽离（Composable 首个落地）

- 新增：`frontend/src/views/security/composables/useWafPolicy.ts`
- 首批迁移能力：
  - policy 列表查询与分页；
  - policy 表单态（新增/编辑）与提交；
  - policy 预览、校验、发布、删除；
  - policy revision 列表与回滚；
  - 默认策略与 CRS 策略选项衍生。

### 2.3 质量门禁

- 每次结构迁移后执行：
  - `pnpm typecheck`
- 保证后端 API path、字段、语义不变。

---

## 3. 非范围（Out of Scope）

- 不调整后端 API 设计；
- 不改变现有业务流程与权限策略；
- 不在本阶段拆分 `caddy.ts`（后续阶段处理）；
- 不做视觉样式重构，仅做结构性重构。

---

## 4. 实施步骤

### Step A：拆分 observe/release/job 为子组件

- 从 `index.vue` 提取三个 tab 的模板与事件绑定；
- 子组件只接收已存在状态与方法，不新增业务分支；
- 先“搬模板”，再“收敛 props”。

### Step B：接入 useWafPolicy

- 在 `index.vue` 引入 `useWafPolicy`；
- 将 policy 相关状态与方法统一由 composable 提供；
- 逐步消除 `index.vue` 中 policy 领域重复逻辑。

### Step C：回归与校验

- 回归关键操作流：
  - 运行策略增删改查；
  - 预览/校验/发布/回滚；
  - Observe 查询与反馈操作；
  - Release 激活/回滚；
  - Job 查询与清理；
- 执行 `pnpm typecheck` 并修正类型漂移。

---

## 5. 风险与应对

### 风险 1：props 数量过多导致可读性下降

- 应对：按领域分组命名（policyStats / policyFeedback / release / job）；
- 后续通过 composable 继续收敛传参与事件分发。

### 风险 2：双向绑定在子组件中行为变化

- 应对：优先使用显式 setter 回调；
- 仅在引用型对象上使用安全透传，避免破坏响应式语义。

### 风险 3：CRS 与 Policy 状态联动回归

- 应对：保留 `index.vue` 中跨域协调入口（如 tab 切换触发刷新）；
- 用最小迁移策略逐步下沉。

---

## 6. 交付与验收标准

满足以下条件即视为本阶段完成：

1. `observe/release/job` 模板从 `index.vue` 抽离完成；
2. `useWafPolicy` 在页面中可用并承接首批 policy 逻辑；
3. 业务行为无变化（手工回归通过）；
4. `pnpm typecheck` 通过；
5. 无 API 合同变更。

---

## 7. 下一阶段建议（Phase 3）

- 继续下沉 composable：`useWafObserve`、`useWafRelease`；
- 抽离 `policy-feedback` 相关状态机；
- 将 `service/api/caddy.ts` 按域拆分并保留兼容导出；
- 补齐 composable 级单元测试与关键交互测试。
