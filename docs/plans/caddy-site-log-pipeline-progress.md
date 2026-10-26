# 站点日志接入：进度与未完成项

> 更新：2026-10-26
> 基线提交：`cdba9d1` feat: 站点日志采集与访问/错误日志分离
> 前置方案：`caddy-site-first-cert-center-plan.md`（站点中心雷池式重构）
> 范围：`backend/{common/ingest,internal/logic/caddy,internal/service,model}`、`frontend/apps/src/{api/caddy,views/caddy}`

---

## 1. 需求背景

站点需要满足三项要求：

| 需求 | 说明 |
|---|---|
| 每站点可开关访问日志 | 关掉后不再产生访问日志文件 |
| 每站点可开关错误日志 | 与访问日志分开控制 |
| 每站点可设单文件大小上限 | 超出后自动处理 |

叠加两条已确认的决策：

- **日志要入库**：站点日志文件必须被采集进 `caddy_logs`，可在面板内查询
- **单文件写入，显示分开**：Caddy 只写一个文件，但面板里访问日志与错误日志分开查看

---

## 2. Caddy 能力边界（方案的技术前提）

这部分是查证 Caddy 官方文档得出的结论，后续改动不应推翻。

### 2.1 `level` 是最低级别，不是白名单

站点块内的 `log` 指令只有一个 `level`，语义是「最低记录级别」，因此：

| 访问日志 | 错误日志 | 可写成的 `level` | 是否可行 |
|---|---|---|---|
| 开 | 开 | `INFO` | ✅ |
| 关 | 开 | `ERROR` | ✅ |
| 关 | 关 | 不写 `import` | ✅ |
| 开 | 关 | — | ❌ **无法表达** |

`level INFO` 必然连带 `ERROR`，Caddy 没有「只放行 INFO、排除 ERROR」的语法。UI 层对此组合禁用并给出说明。

### 2.2 `roll_size` 是滚动归档，不是截断覆盖

写满阈值后，Caddy 把当前文件改名为 `<name>-<时间戳>-size.log` 归档，再新建同名文件继续写。

- 精度 **1MB**，`1.5MB` 会向上取整为 `2MB`，UI 用整数 MB
- `roll_keep 0` 被 Caddy 当成未设置，回落默认 **10**
- `roll_keep 1` = 1 个活动文件 + 1 个归档文件，是「文件数最少」的下限

**Caddy 没有「超限清空重写」模式。** 真正单文件只能由后端自行轮询清空，会与 Caddy 写入竞争并丢日志，因此不采用。

### 2.3 错误日志无法按站点隔离命名空间

站点块内的 `log` 只控制访问日志；全局 `log` 控制运行时日志。错误日志的 logger 命名空间是模块级（`http.log`、`tls` 等），没有「某站点的错误日志」这一独立命名空间。

**因此「分开」只能在采集与查询层做**：Caddy 用 `level INFO` 把访问日志和错误日志写进同一个文件，LogFlux 采集时按 JSON 里的 `level` / `logger` 字段分流。

---

## 3. 已完成

### 3.1 采集链路

| 项 | 实现 | 位置 |
|---|---|---|
| 归档文件排除 | `IsActiveLogFileName` 排除 `-size` / `-time` 后缀（含 `.gz`），正则要求完整时间戳，避免误杀 `file-size.log` | `common/ingest/shared/common.go` |
| inode 校验 | `LogIngestCursor.FileID` 记录文件 inode，文件重建后从 0 重采而非读到错位数据 | `model/ingest/log_ingest_cursor.go`、`common/ingest/caddy/ingestor.go` |
| 目录源注册 | `EnsureSiteLogSource` 把 `/var/log/caddy` 注册为目录源，新站点文件自动发现 | `common/ingest/site_log_source.go` |
| 启动时加载 | 启动即注册目录源并加载所有 `type='caddy'` 的启用源，不必等配置变更 | `internal/svc/service_context.go` |
| 按节点隔离 | `LogSource.ServerID` + `(path, server_id)` 复合唯一索引，`syncCaddyLogSources` 按节点过滤 | `model/ingest/log_source.go`、`internal/logic/caddy/caddy_helpers.go` |

> 归档排除的正则曾因 `-(?:size|time)\.log$` 过宽把 `file-size.log` 误判为归档，导致真实日志漏采。已收紧为要求完整时间戳 `-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}(?:\.\d+)?-(?:size|time)\.log(?:\.gz)?$`，并有单测锁定。

### 3.2 level / logger 字段

```go
// model/caddy/caddy_log.go
Level  string `gorm:"size:16;index:idx_level_log_time,priority:1"`
Logger string `gorm:"size:128;index:idx_logger_log_time,priority:1"`
```

`CaddyLogArchive` 同步加字段，归档不丢数据。

三条解析路径均覆盖：

- JSON 访问日志 → 读实际 `level` / `logger`
- Coraza 审计日志 → 固定 `info` / `http.log.access`
- console 文本格式 → 固定 `info` / `http.log.access`

顺手修复：`LogTime` 补上 `index:idx_host_log_time,priority:1`，此前该复合索引因缺这一处声明退化为 `host` 单列索引。

### 3.3 查询分流

`CaddyLogQuery.LogKind` 取值 `access` / `error`：

```sql
-- access
logger = 'http.log.access'
-- error：双条件兜底，避免 Caddy 调整命名空间后错误日志整体消失
(level IN ('error','fatal','panic')) OR (logger <> '' AND logger <> 'http.log.access')
```

API 层 `CaddyLogReq.logKind` 已接通，`CaddyLogItem` 返回 `level` / `logger`。

### 3.4 日志模板（snippet）管理

复用 `entry-block` 的存储与合并模式：

| 项 | 内容 |
|---|---|
| 存储键 | `system_configs` 里的 `caddy_log_snippet` |
| 内置默认 | `//go:embed default_log_snippet.caddyfile` |
| 合并 | `mergeCaddyLogSnippet`，在 `mergeCaddyManagedBlocks` 中与 `mergeGlobalOptions`、`mergeEntryBlock` 同链 |
| 路由 | `GET /caddy/log-snippet`、`PUT /caddy/log-snippet`、`POST /caddy/log-snippet/reset-default` |

模板形态（带键名的块参数，Caddy 2.9+）：

```caddyfile
(subdomain-log) {
  log {
    output file /var/log/caddy/{args.name}.log {
      roll_size {args.size}mb
      roll_keep 1
    }
    level {args.level}
    format json
  }
}
```

站点侧：

```caddyfile
import subdomain-log {
  name myapp
  size 10
  level INFO
}
```

采用带键名的块参数而非位置参数 `{args[0]}`，避免将来加参数要改所有站点。

### 3.5 前端逻辑层

| 文件 | 职责 |
|---|---|
| `views/caddy/config/site-log-settings.ts` | 参数模型、`settingsToLevel` / `levelToSettings` 四象限映射、snippet 与 import 块渲染、`parseSiteLogImport`（兼容旧的单行写法）、`validateSiteLogSettings` |
| `views/caddy/config/site-capabilities.ts` | `applySiteLogToSiteBlock`：能删整块 `import subdomain-log {...}` 并重写多行，保留缩进风格与 CRLF，幂等 |
| `views/caddy/sites/use-site-detail-edit.ts` | `logSettings` 状态、`logErrorForced`（访问日志开则强制错误日志开）、`logInlineConflict`（检测内联 `log` 指令）、advanced scope 保存时调用 `applySiteLogToSiteBlock` |
| `api/caddy/log-snippet.ts` | 三个接口封装 |

`SiteDetailTab` 类型已扩展为 `'basic' | 'protect' | 'advanced' | 'logs'`。

### 3.6 站点列表与详情（前序审查的修复）

| 项 | 说明 |
|---|---|
| 指标 SQL 合并 | 4 条聚合查询合并为 1 条 `COUNT(*) FILTER` 条件聚合 |
| 多域名指标 | `siteMetricsForDomains` 汇总站点全部域名，不再只统计主域名 |
| diff 弹窗 | 改用 `buildLineDiff` 渲染真实行级 diff，含删除行；`unchangedLineCount` 改为「块外行数」 |
| 4xx/5xx 展示 | 详情页头部新增两项，卡片在 5xx > 0 时显示告警标签 |
| 死分支清理 | 删除两处 `v-if="!site.enabled"`（parser 恒写 `enabled: true`，标签永不显示） |
| 废弃组件 | 删除 `SiteDrawer.vue`（557 行零引用） |

---

## 4. 未完成

### 4.1 P0：站点详情「日志」Tab 视图缺失

**这是当前最明显的缺口。** 逻辑层与 API 层全部就绪，但模板里没有渲染。

现状：

- `SiteDetailTab` 已含 `'logs'`
- `Tabs` 里只有 `basic` / `protect` / `advanced` 三个 `TabPane`（`SiteDetail.vue` 378-581 行）
- `getCaddyLogsApi` 已支持 `logKind` 参数（`api/caddy/server.ts:142`）
- 无任何视图引用这些能力

需要补：

```vue
<TabPane key="logs" tab="日志">
  <!-- RadioButton 切换 访问日志 / 错误日志 -->
  <!-- 共用表格：getCaddyLogsApi({ host, logKind, page, pageSize, startTime, endTime }) -->
  <!-- 错误日志视图额外显示 level、logger 两列 -->
</TabPane>
```

该 Tab 只读，不参与保存。

### 4.2 P0：高级 Tab 的日志三开关未渲染

`use-site-detail-edit.ts` 里 `logSettings` / `logLevel` / `logLevelLabel` / `logErrorForced` / `logInlineConflict` / `logSizeMinMb` / `logSizeMaxMb` 全部已导出，但 `SiteDetail.vue` 的 advanced Tab（537-581 行）没有对应控件。

需要补：

- 开关「访问日志」「错误日志」
- 数字输入「单文件上限（MB）」，范围取 `logSizeMinMb` / `logSizeMaxMb`
- `logErrorForced` 为 true 时禁用错误日志开关并给 Tooltip
- 只读展示 `logLevelLabel`
- 保存走现有 `requestSave('advanced')`，保存链路已在 composable 内接好

### 4.3 P1：全局设置的「日志模板」卡片缺失

`api/caddy/log-snippet.ts` 已定义三个接口，但 `views/caddy/settings/` 下无任何引用（只有 `api/caddy/index.ts` 做了 re-export）。

需要补一张卡片：snippet 原文 textarea、恢复默认按钮、保存后提示「模板变更需重新应用站点配置才生效」。

### 4.4 P1：历史数据 `level` / `logger` 为空

`level` / `logger` 是新增列，AutoMigrate 会加列但历史行为空串。

影响：

- 错误日志视图筛不到历史数据（`logger <> ''` 条件把空串排除）
- 访问日志视图同样筛不到历史数据（`logger = 'http.log.access'` 不匹配空串）

三种处理方向，尚未决策：

| 方案 | 说明 | 代价 |
|---|---|---|
| A. 接受空值 | 新数据才有分流能力，文档说明 | 零成本，但历史查询体验断层 |
| B. 按 `raw_log` 回填 | 从 `raw_log` JSON 解析 `level` / `logger` 回填 | 需处理非 JSON 的老格式；大批量 UPDATE |
| C. 查询兼容空值 | `access` 条件放宽为 `logger = 'http.log.access' OR logger = ''` | 会混入历史错误日志，语义不纯 |

倾向 A + C 结合：历史数据按「未知类别」单独处理，不污染 access/error 两个视图。

### 4.5 P1：现有 snippet 迁移

当前生产 Caddyfile 里手写的 snippet 是：

```caddyfile
(subdomain-log) {
  log {
    output file /var/log/caddy/{args[0]}.log {
      roll_size 10MB
      roll_keep 10
      roll_keep_for 7d
    }
    level debug
    format json
  }
}
```

与 LogFlux 托管的新模板差异：

| 项 | 现有 | 新模板 | 影响 |
|---|---|---|---|
| 参数形式 | `{args[0]}` 位置参数 | `{args.name}` 键名参数 | 站点侧 `import` 写法需改 |
| `roll_keep` | 10 | 1 | 归档文件从最多 10 个降到 1 个 |
| `roll_keep_for` | 7d | 移除 | `roll_keep 1` 已足够，两者取更严格者 |
| `level` | `debug` | 参数化 `INFO` | 不再记录 debug |

**迁移是行为变更，会影响现有所有站点。** 需确认：

1. 是否保留现有站点行为，只对新站点生效
2. 还是统一迁移，接受短期的日志量变化

注：日志已入库，磁盘上的文件只是采集缓冲，`roll_keep 1` 不会丢失可查询的历史数据。

### 4.6 P2：内联 `log` 指令的冲突处理

`logInlineConflict` 已能检测站点块里手写的 `log { }`。此时三开关不应生效（会与内联指令冲突）。

UI 层需在此情况下禁用控件并提示「检测到内联 log 指令，请在原文模式维护」。目前检测逻辑在，提示未接。

---

## 5. 验证基线

```
后端：go build ./... && go vet ./... && go test ./...        全绿
前端：npx vue-tsc --noEmit --skipLibCheck                    零错误
      npx vitest run src/views/caddy                         200 用例通过
```

前端有 5 个既有失败文件（`useCaddyConfigIO` / `useCaddyServers` / `useConfigHistory` / `useSimpleWaf` / `useSiteMetrics`），原因是 `dayjs/plugin/advancedFormat` 模块解析失败，属于环境问题，与本次改动无关，改动前基线即失败。

新增测试：

| 文件 | 用例数 |
|---|---|
| `site-log-settings.test.ts` | 31 |
| `site-capabilities.test.ts` | 36（含既有 + 本次新增的块级写入用例） |
| `site-detail-diff.test.ts` | 4 |
| `log_snippet_merge_test.go` | 覆盖插入/替换/无全局块/幂等/CRLF |
| `common_test.go` | 归档文件名识别 |
| `log_model_breakdown_test.go` | 单条 SQL 聚合语义 |

---

## 6. 数据库变更

`caddy_logs` 与 `caddy_logs_archive` 新增 `level`、`logger` 两列，`log_ingest_cursors` 新增 `file_id`，`log_sources` 新增 `server_id` 且唯一索引改为 `(path, server_id)`。

均由 GORM AutoMigrate 处理。**注意**：`log_sources` 的唯一索引变更在已有表上可能需要手工处理——旧索引 `path` 单列唯一会与新复合索引并存，若此前存在跨节点同路径数据，迁移时可能冲突。

升级前建议先检查：

```sql
SELECT path, COUNT(*) FROM log_sources GROUP BY path HAVING COUNT(*) > 1;
```

---

## 7. 后续优先级建议

| 优先级 | 事项 | 依据 |
|---|---|---|
| P0 | 站点详情「日志」Tab（4.1） | 逻辑已完备，只差视图，收益最直接 |
| P0 | 高级 Tab 三开关（4.2） | 同上 |
| P1 | 全局设置日志模板卡片（4.3） | 无 UI 则模板只能走 API 改 |
| P1 | snippet 迁移决策（4.5） | 阻塞存量站点上线新能力 |
| P2 | 历史数据空值处理（4.4） | 不影响新数据 |
| P2 | 内联 log 冲突提示（4.6） | 边界场景 |
