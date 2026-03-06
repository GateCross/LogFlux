# LogFlux 安全管理页面结构映射说明

> 对应方案文档：`docs/plans/security-management-optimization-execution-plan.md`
> 对应任务清单：`docs/plans/security-management-optimization-task-checklist.md`
> 目的：明确当前 `security` 模块的旧 route / menu / tab 结构与目标一级领域结构之间的映射关系，作为后续壳层化和导航 schema 收敛的实施依据。

## 1. 文档目标

当前 `/security` 模块已经具备完整能力，但前端导航模型同时存在：

- 一级菜单路由
- 隐藏子路由
- 页面内部 tab
- 基于 query 的 legacy tab 兼容

这些概念在 `frontend/src/views/security/index.vue` 与 `frontend/src/store/modules/route/index.ts` 中同时存在，导致后续调整结构时容易出现：

- 入口语义不一致
- 路由与页面状态不同步
- 旧链接兼容行为不明确
- 新旧信息架构映射缺少统一基线

本文件用于解决以上问题，明确“当前是什么、目标变成什么、过渡期怎么兼容”。

---

## 2. 当前结构概览

## 2.1 当前一级菜单组

当前在路由和页面层面实际对外暴露的一级菜单组有 4 个：

1. `security_source`
2. `security_policy`
3. `security_observe`
4. `security_ops`

它们在前端页面中对应的菜单分组语义分别是：

- `source`
- `policy`
- `observe`
- `ops`

## 2.2 当前页面内部 tab

当前页面内部存在 8 个实际业务 tab：

1. `source`
2. `runtime`
3. `crs`
4. `exclusion`
5. `binding`
6. `observe`
7. `release`
8. `job`

其中：

- `source`、`observe` 本身既是菜单组也是具体 tab
- `policy` 菜单组下实际包含 4 个 tab：`runtime / crs / exclusion / binding`
- `ops` 菜单组下实际包含 2 个 tab：`release / job`

## 2.3 当前路由层结构

当前 `security` 路由族包含以下子路由：

- `security_source`
- `security_policy`
- `security_observe`
- `security_ops`
- `security_runtime`
- `security_crs`
- `security_exclusion`
- `security_binding`
- `security_release`
- `security_job`

其中：

- 前 4 个是菜单可见入口
- 后 6 个是隐藏路由，用于兼容深链和页面状态定位

---

## 3. 当前结构映射表

## 3.1 当前菜单组 -> 当前 tab

| 当前菜单组 | 当前 route name | 当前包含 tab | 当前语义 |
| --- | --- | --- | --- |
| `source` | `security_source` | `source` | 更新源与引擎检查 |
| `policy` | `security_policy` | `runtime` / `crs` / `exclusion` / `binding` | 运行策略相关配置 |
| `observe` | `security_observe` | `observe` | 策略观测与反馈 |
| `ops` | `security_ops` | `release` / `job` | 发布与任务运维 |

## 3.2 当前隐藏子路由 -> 当前 tab

| 当前隐藏 route name | path | 对应 tab | 说明 |
| --- | --- | --- | --- |
| `security_runtime` | `/security/runtime` | `runtime` | policy 域默认子能力 |
| `security_crs` | `/security/crs` | `crs` | policy 域 CRS 调优 |
| `security_exclusion` | `/security/exclusion` | `exclusion` | policy 域规则例外 |
| `security_binding` | `/security/binding` | `binding` | policy 域策略绑定 |
| `security_release` | `/security/release` | `release` | ops 域版本发布 |
| `security_job` | `/security/job` | `job` | ops 域任务日志 |

## 3.3 当前页面状态模型

当前页面通过以下两个核心状态驱动导航：

- `activeMenu`
- `activeTab`

它们的关系是：

- `activeMenu` 决定当前属于哪一个菜单组
- `activeTab` 决定当前页面实际展示哪个业务域内容
- `activeTab` 与路由、query(`activeTab`) 存在双向兼容关系

---

## 4. 目标结构概览

建议将当前安全管理的导航与页面结构收束为 4 个一级领域：

1. `规则来源`
2. `策略中心`
3. `观测与处置`
4. `发布运维`

对应英文领域标识建议为：

- `source`
- `policy`
- `observe`
- `ops`

这 4 个领域与当前 4 个一级菜单组在数量上保持一致，但含义更加稳定，后续不再强调页面上所有内部 tab 都是平级入口。

---

## 5. 当前结构 -> 目标结构映射

## 5.1 一级菜单组映射

| 当前菜单组 | 当前语义 | 目标一级领域 | 是否保留 route name | 调整说明 |
| --- | --- | --- | --- | --- |
| `source` | 更新源配置 | `规则来源` | 是 | 保留入口，但强调“来源供应链”而不是单纯 source 列表 |
| `policy` | 策略配置集合 | `策略中心` | 是 | 保留入口，但不再把 runtime/crs/exclusion/binding 作为同级主入口 |
| `observe` | 观测与反馈 | `观测与处置` | 是 | 保留入口，但在领域内拆分分析与处置视图 |
| `ops` | 版本与任务 | `发布运维` | 是 | 保留入口，突出发布与审计语义 |

## 5.2 当前 tab -> 目标一级领域映射

| 当前 tab | 当前名称 | 目标一级领域 | 目标二级位置 |
| --- | --- | --- | --- |
| `source` | 更新源配置 | `规则来源` | 规则来源主页 |
| `runtime` | 运行模式 | `策略中心` | 策略工作区 / 基础设置 |
| `crs` | CRS 调优 | `策略中心` | 策略工作区 / CRS 调优 |
| `exclusion` | 规则例外 | `策略中心` | 策略工作区 / 规则例外 |
| `binding` | 策略绑定 | `策略中心` | 策略工作区 / 作用域绑定 |
| `observe` | 策略观测 | `观测与处置` | 领域首页，内部拆为分析/处置 |
| `release` | 版本发布管理 | `发布运维` | 发布管理 |
| `job` | 任务日志 | `发布运维` | 任务审计 |

## 5.3 当前隐藏子路由 -> 目标结构映射

| 当前隐藏 route | 目标一级领域 | 目标内部定位方式 | 兼容策略 |
| --- | --- | --- | --- |
| `security_runtime` | `策略中心` | 打开策略工作区默认视图 | 保留 route name，内部落到基础设置 |
| `security_crs` | `策略中心` | 打开策略工作区 CRS 区块 | 保留 route name，内部落到 CRS 调优 |
| `security_exclusion` | `策略中心` | 打开策略工作区例外区块 | 保留 route name，内部落到规则例外 |
| `security_binding` | `策略中心` | 打开策略工作区绑定区块 | 保留 route name，内部落到策略绑定 |
| `security_release` | `发布运维` | 打开发布管理视图 | 保留 route name，内部定位到 release |
| `security_job` | `发布运维` | 打开任务日志视图 | 保留 route name，内部定位到 job |

---

## 6. 推荐的目标页面结构

## 6.1 目标页面层级

建议结构如下：

1. `SecurityShell`
   - 负责一级领域切换
   - 负责旧路由兼容解析
   - 负责标题与导航状态同步
   - 负责共享弹窗挂载与领域间刷新协调

2. `SecuritySourcePage`
   - source 领域容器

3. `SecurityPolicyPage`
   - policy 领域容器
   - 内部以策略工作区组织内容

4. `SecurityObservePage`
   - observe 领域容器
   - 内部分为“效果分析”和“误报处置”

5. `SecurityOpsPage`
   - ops 领域容器
   - 内部分为 release / job

## 6.2 策略中心内部结构

建议将原先 4 个平级 tab：

- `runtime`
- `crs`
- `exclusion`
- `binding`

重构为一个策略工作区的 4 个配置区块：

- 基础设置
- CRS 调优
- 规则例外
- 策略绑定

同时保留：

- 指令预览
- 校验
- 发布
- Revision 历史

## 6.3 Observe 内部结构

建议将原 `observe` 拆为两个内部子视图：

### A. 效果分析

- summary
- trend
- top hosts / paths / methods
- drilldown
- export

### B. 误报处置

- feedback list
- batch process
- assignee / SLA
- exclusion draft generation

## 6.4 发布运维内部结构

建议将 `ops` 领域稳定为两个内部子视图：

- release 管理
- job 审计

这个结构与当前行为一致，仅明确为领域内二级结构，而不再混在页面 tab 体系中。

---

## 7. 路由兼容策略

## 7.1 兼容目标

后续重构应保证以下能力不丢失：

- 旧菜单入口仍可访问
- 旧隐藏子路由仍可访问
- query 中的 legacy `activeTab` 仍可解析
- Observe 相关 query 可继续用于状态恢复

## 7.2 推荐兼容规则

### 规则 1：route 优先于 legacy query

如果当前 route name 已明确指向某个领域或内部子视图，则优先按 route 解析，不再依赖 `activeTab` 推断。

### 规则 2：legacy `activeTab` 仅作为兼容输入

后续新的导航模型中，`activeTab` 不应继续作为主要页面状态来源，仅作为旧链接兼容来源。

### 规则 3：Observe 查询参数单独保留

Observe 的 `policyId / window / intervalSec / topN / host / path / method` 仍保留 query 同步能力，但应与一级领域切换逻辑解耦。

### 规则 4：隐藏子路由继续有效，但内部转换为领域内定位

如：

- `security_crs` -> 打开 `策略中心` 并定位到 CRS 区块
- `security_job` -> 打开 `发布运维` 并定位到 Job 视图

---

## 8. 实施建议

## 8.1 文档先行

在代码重构前，先以本映射文档作为统一基线，避免边做边改语义。

## 8.2 先抽 schema，再改壳层

建议先抽一份统一的 security navigation schema，承载：

- 一级领域定义
- 当前 route name
- legacy tab
- 默认子视图
- 标题
- 是否显示二级导航

然后再让 `index.vue` 退化为 `SecurityShell`。

## 8.3 保持 route name 稳定

本轮优化优先保证兼容，不建议先改 route name；若未来要进一步收敛路由，也应在当前一级领域结构稳定后再评估。

---

## 9. 结论

当前 `security` 模块最复杂的不是单一功能，而是导航模型同时承担了“菜单、tab、隐藏子路由、legacy query 兼容”四类职责。

后续优化的关键不是简单删 tab，而是：

- 先用统一 schema 收敛导航语义
- 再把页面改造成壳层 + 领域容器
- 再把策略中心与 Observe 这两个高复杂度区域做内部结构收束

本映射文档用于保证旧结构到新结构的迁移过程中，所有入口、深链、标题、默认页和兼容行为都有明确落点。
