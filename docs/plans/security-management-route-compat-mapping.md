# 安全管理路由兼容映射说明

> 更新时间：2026-03-06
> 对应模块：`/security`
> 关联文件：
> - `frontend/src/views/security/navigation.ts`
> - `frontend/src/views/security/composables/useSecurityNavigation.ts`
> - `frontend/src/router/elegant/transform.ts`

## 1. 文档目的

本文用于说明安全管理模块在完成“4 个一级领域入口”重构后，旧路由、旧 deep link、旧 tab 语义如何继续兼容。

目标：

- 明确新的标准入口；
- 明确旧路由仍如何映射到新领域；
- 明确 `activeTab` 与 Observe query 的兼容规则；
- 为后续排障、回归和文档同步提供统一依据。

---

## 2. 新的标准入口

当前安全管理的一级领域入口统一为 4 个：

| 一级领域 | 路由名 | 路径 | 默认子视图 |
| --- | --- | --- | --- |
| 规则来源 | `security_source` | `/security/source` | `source` |
| 策略中心 | `security_policy` | `/security/policy` | `runtime` |
| 观测与处置 | `security_observe` | `/security/observe` | `observe` |
| 发布运维 | `security_ops` | `/security/ops` | `release` |

说明：

- 这 4 个入口是当前推荐的标准入口；
- 页面一级导航、壳层高亮、领域刷新逻辑都围绕这 4 个入口展开；
- 对用户而言，已不再强调 `runtime/crs/exclusion/binding/release/job` 是并列一级页面。

---

## 3. 兼容保留的旧路由

为了兼容旧书签、旧跳转和历史 deep link，以下旧路由名与路径继续保留：

| 旧语义 | 路由名 | 路径 | 归属新领域 |
| --- | --- | --- | --- |
| 更新源配置 | `security_source` | `/security/source` | `source` |
| 运行模式 | `security_runtime` | `/security/runtime` | `policy` |
| CRS 调优 | `security_crs` | `/security/crs` | `policy` |
| 规则例外 | `security_exclusion` | `/security/exclusion` | `policy` |
| 策略绑定 | `security_binding` | `/security/binding` | `policy` |
| 策略观测 | `security_observe` | `/security/observe` | `observe` |
| 发布运维总入口 | `security_ops` | `/security/ops` | `ops` |
| 版本发布管理 | `security_release` | `/security/release` | `ops` |
| 任务日志 | `security_job` | `/security/job` | `ops` |

说明：

- 这些旧路由不会被当作新的一级信息架构继续扩展；
- 它们的主要职责是保持兼容，并在进入页面后自动折叠到新的领域模型下；
- 领域归属关系由 `frontend/src/views/security/navigation.ts` 中的 `SECURITY_ROUTE_NAME_MENU_MAP` 统一维护。

---

## 4. Tab 到领域的兼容映射

当前系统仍保留旧 tab key，用于兼容旧交互和旧 query：

| tab key | 对应领域 | 说明 |
| --- | --- | --- |
| `source` | `source` | 规则来源唯一子视图 |
| `runtime` | `policy` | 策略中心默认子视图 |
| `crs` | `policy` | 策略中心子视图 |
| `exclusion` | `policy` | 策略中心子视图 |
| `binding` | `policy` | 策略中心子视图 |
| `observe` | `observe` | 观测与处置唯一主视图 |
| `release` | `ops` | 发布运维默认子视图 |
| `job` | `ops` | 发布运维子视图 |

兼容关系来源：

- `frontend/src/views/security/navigation.ts` 中的 `SECURITY_TAB_MENU_MAP`

---

## 5. 路由解析规则

## 5.1 入口解析优先级

当前路由状态恢复遵循以下优先级：

1. 优先根据当前 `route.name` 判断所属一级领域；
2. 如果 `route.name` 无法命中，则回退读取 query 中的 `activeTab`；
3. 若二者都无法识别，则回退到 `source`。

对应实现：

- `resolveSecurityMenuFromRoute()`：`frontend/src/views/security/navigation.ts`
- `syncNavigationStateFromRoute()`：`frontend/src/views/security/composables/useSecurityNavigation.ts`

## 5.2 子视图解析规则

进入某个领域后，子视图解析规则如下：

- 若 query 中 `activeTab` 属于当前领域允许的 tab，则使用该 tab；
- 否则使用该领域的默认子视图；
- 默认子视图定义在 `SECURITY_MENU_SCHEMA` 中。

默认子视图如下：

| 领域 | 默认 tab |
| --- | --- |
| `source` | `source` |
| `policy` | `runtime` |
| `observe` | `observe` |
| `ops` | `release` |

---

## 6. 跳转生成规则

`navigateToSecurityTab()` 负责统一生成安全管理内部跳转。

规则如下：

### 6.1 Source / Policy / Ops

- 跳到某个领域的默认子视图时，不写入 `activeTab`；
- 跳到某个领域的非默认子视图时，在 query 中写入 `activeTab`；
- 目标路由名使用对应领域的标准入口路由名，而不是旧子路由名。

示例：

| 目标 tab | 最终路由名 | query |
| --- | --- | --- |
| `source` | `security_source` | 无 |
| `runtime` | `security_policy` | 无 |
| `crs` | `security_policy` | `activeTab=crs` |
| `exclusion` | `security_policy` | `activeTab=exclusion` |
| `binding` | `security_policy` | `activeTab=binding` |
| `release` | `security_ops` | 无 |
| `job` | `security_ops` | `activeTab=job` |

### 6.2 Observe

Observe 不使用 `activeTab` 组织一级兼容，而是保留分析上下文 query：

- `policyId`
- `window`
- `intervalSec`
- `topN`
- `host`
- `path`
- `method`

对应常量：

- `SECURITY_OBSERVE_QUERY_KEYS`：`frontend/src/views/security/navigation.ts`

说明：

- 在 Observe 内部切换或刷新时，这些 query 会被保留，用于恢复 drilldown 和筛选上下文；
- 这也是旧观察类 deep link 仍可正常恢复状态的关键兼容点。

---

## 7. 兼容映射总表

| 访问方式 | 示例 | 最终领域 | 最终 activeTab | 兼容说明 |
| --- | --- | --- | --- | --- |
| 新标准入口 | `/security/source` | `source` | `source` | 标准入口 |
| 新标准入口 | `/security/policy` | `policy` | `runtime` | 标准入口 |
| 新标准入口 | `/security/observe` | `observe` | `observe` | 标准入口 |
| 新标准入口 | `/security/ops` | `ops` | `release` | 标准入口 |
| 旧子路由 | `/security/runtime` | `policy` | `runtime` | 直接兼容 |
| 旧子路由 | `/security/crs` | `policy` | `crs` | 直接兼容 |
| 旧子路由 | `/security/exclusion` | `policy` | `exclusion` | 直接兼容 |
| 旧子路由 | `/security/binding` | `policy` | `binding` | 直接兼容 |
| 旧子路由 | `/security/release` | `ops` | `release` | 直接兼容 |
| 旧子路由 | `/security/job` | `ops` | `job` | 直接兼容 |
| 标准入口 + legacy query | `/security/policy?activeTab=crs` | `policy` | `crs` | query 兼容 |
| 标准入口 + legacy query | `/security/ops?activeTab=job` | `ops` | `job` | query 兼容 |
| Observe 深链 | `/security/observe?policyId=1&window=24h&host=a.com` | `observe` | `observe` | query 上下文保留 |

---

## 8. 回归验证建议

补充回归时，至少覆盖以下兼容入口：

1. `/security/policy`
2. `/security/policy?activeTab=crs`
3. `/security/runtime`
4. `/security/crs`
5. `/security/exclusion`
6. `/security/binding`
7. `/security/ops`
8. `/security/ops?activeTab=job`
9. `/security/release`
10. `/security/job`
11. `/security/observe?policyId=1&window=24h&host=example.com`

验证点：

- 一级领域高亮正确；
- 页面标题正确；
- 子视图正确恢复；
- 浏览器前进/后退不丢状态；
- Observe query 能恢复筛选与 drilldown；
- 从旧 deep link 进入后，页面操作仍走当前新结构。

---

## 9. 当前结论

当前安全管理路由兼容策略已经形成稳定模型：

- 外部推荐入口统一收敛到 4 个标准领域路由；
- 内部仍保留旧子路由名和旧 tab 语义作为兼容层；
- Observe 继续保留 query 级状态恢复能力；
- 导航、标题、默认子视图、deep link 恢复都已由统一 schema 驱动。

这意味着后续即使继续优化页面结构，也不需要再次散落维护多套兼容规则，只需继续围绕 `navigation.ts` 的统一 schema 演进即可。
