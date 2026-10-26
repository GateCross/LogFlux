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

`CaddyLogArchive` 已随归档机制一并移除（见第 7 节）。

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

模板形态（位置参数；Caddy 官方文档确认 snippet 参数只有 `{args[n]}` 位置形式，没有 `{args.name}` 键名形式）：

```caddyfile
(subdomain-log) {
  log {
    output file /var/log/caddy/{args[0]}.log {
      roll_size {args[1]}mb
      roll_keep 1
    }
    level {args[2]}
    format json
  }
}
```

站点侧：

```caddyfile
import subdomain-log myapp 10 INFO
```

> 三个参数依次为 name / size / level。曾一度改用带键名的块参数（`{args.name}`），
> 后经 Caddy 官方文档确认 snippet 参数只有位置形式，键名写法在 Caddy 中整体失效，
> 已回退为位置参数；存量站点侧的旧写法由 `migrateSiteLogImports` 在每次应用配置时自动迁移（见 4.5）。

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

## 4. 未完成项收尾（已全部完成）

### 4.1 ✅ 站点详情「日志」Tab 视图

前端已补齐：`SiteDetail.vue` 新增 `logs` TabPane，访问日志 / 错误日志 RadioButton 切换，
共用表格走 `getCaddyLogsApi({ host, logKind, ... })`，错误日志视图额外显示 `level` / `logger` 两列。只读，不参与保存。

### 4.2 ✅ 高级 Tab 的日志三开关

前端已补齐：访问日志 / 错误日志开关、单文件上限（MB）数字输入（范围取 `logSizeMinMb` / `logSizeMaxMb`）、
`logErrorForced` 时禁用错误日志开关并给 Tooltip、只读展示 `logLevelLabel`；保存走现有 `requestSave('advanced')`。

### 4.3 ✅ 全局设置的「日志模板」卡片

前端已补齐：`views/caddy/settings/components/SettingsSiteLogCard.vue` 提供 snippet 原文编辑与恢复默认，
同卡片带出保留策略设置；保存后提示「模板变更需重新应用站点配置才生效」。

### 4.4 ✅ 历史数据 `level` / `logger` 为空

采用方案 B：启动期单条 SQL 分批回填（`common/ingest/caddy/log_field_backfill.go`）。

- 判定规则：`raw_log` 为 JSON 字符串标量的是旧 console 文本行（按访问日志补 `info` / `http.log.access`）；
  含 `transaction` 键的是 Coraza 审计行（同上）；其余对象行从 `raw_log` 取 `level` / `logger`，取不到按访问日志兜底。
- 每批 `LIMIT 1000`，`UPDATE ... FROM` 推进到 `RowsAffected` 为 0；全部完成后置位 `system_configs` 键
  `caddy_log_field_backfill_done`，中途失败不置位，下次启动重跑（幂等）。
- jsonb 的 `?` 操作符与 GORM 占位符冲突，JSON 判断一律用 `jsonb_exists` 函数形式。

### 4.5 ✅ 现有 snippet 迁移

决策：统一迁移，模板与站点侧写法全部改为位置参数。原因：Caddy 官方文档确认 snippet 参数只有
`{args[n]}` 位置形式，键名参数 `{args.name}` 不存在，原模板与站点侧块参数写法在 Caddy 中整体失效。

迁移规则（`internal/logic/caddy/log_snippet_migrate.go`，在 `mergeCaddyManagedBlocks` 合并 snippet 之后自动执行）：

| 存量形态 | 迁移结果 |
|---|---|
| 单行 `import subdomain-log <name>` | 补为 `import subdomain-log <name> 10 DEBUG`（存量站点原本 level debug，保持日志量不骤减） |
| 多行块 `import subdomain-log { name X size Y level Z }` | 整块替换为单行 `import subdomain-log X Y Z` |
| 已经是 3 参数的单行 | 不动 |

幂等，保留行缩进与 CRLF；块内缺 `size` / `level` 按 `10` / `DEBUG` 补齐，缺 `name` 时整块保持原样。
默认 `:80` 入口块模板同步写为 `import subdomain-log local 10 INFO`。

### 4.6 ✅ 内联 `log` 指令的冲突处理

前端已接提示：检测到内联 `log { }` 时禁用三开关控件并提示「检测到内联 log 指令，请在原文模式维护」。

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
| `log_snippet_migrate_test.go` | 覆盖单行 1 参/块形式/已迁移/缩进 CRLF/幂等/多站点/同名前缀 |
| `common_test.go` | 归档文件名识别 |
| `log_model_breakdown_test.go` | 单条 SQL 聚合语义 |

---

## 6. 数据库变更

`caddy_logs` 新增 `level`、`logger` 两列，`log_ingest_cursors` 新增 `file_id`，`log_sources` 新增 `server_id` 且唯一索引改为 `(path, server_id)`。

均由 GORM AutoMigrate 处理。**注意**：`log_sources` 的唯一索引变更在已有表上可能需要手工处理——旧索引 `path` 单列唯一会与新复合索引并存，若此前存在跨节点同路径数据，迁移时可能冲突。

升级前建议先检查：

```sql
SELECT path, COUNT(*) FROM log_sources GROUP BY path HAVING COUNT(*) > 1;
```

## 7. 7 天日志保留

站点日志的保留策略以 `system_configs` 为唯一真值，接口与清理任务统一从 `model/system/log_retention.go` 读写，缺省 7 天。

| 项 | 内容 |
|---|---|
| 存储键 | `caddy_log_retention`，值为 `{"enabled":bool,"days":int}` |
| 缺省 | 键不存在、JSON 解析失败、days 越界（<1 或 >3650）时按 `{enabled:true, days:7}` 处理 |
| 开关 | `enabled=false` 时清理任务跳过本轮，不删除任何日志；查询接口照常返回当前配置 |
| 删除语义 | 每天凌晨 2 点删除 `log_time` 早于「现在 − days」的 `caddy_logs` 行；单批 2000 行、单轮最多 500 批，单轮删不完的留待下一轮 |

配套变更与升级注意事项：

- 清理任务（`internal/tasks/log_cleanup.go`）**无条件启动**，是否清理由保留策略决定；不再受配置文件 `Archive.Enabled` 门控。
- 旧 `Archive` 配置段（`Enabled` / `RetentionDay` / `ArchiveTable`）已废弃，字段保留只为旧配置文件解析不报错，运行时不读取。
- 归档机制整体移除：归档表 `caddy_logs_archive`、存储过程 `archive_old_logs()`、`CaddyLogArchive` 结构体均已删除。**数据库里的旧表与旧函数不会自动删除**，如需清理可手工执行：
  ```sql
  DROP TABLE IF EXISTS caddy_logs_archive;
  DROP FUNCTION IF EXISTS archive_old_logs(INTEGER);
  ```
- 清理是破坏性路径：读库失败时跳过本轮并告警，**不回落默认天数**（`LoadLogRetentionStrict`），与查询接口的宽容读取刻意不同。
- 历史日志的 `level` / `logger` 空值由启动期回填补齐（见 4.4），完成标记 `caddy_log_field_backfill_done` 置位后不再重跑。

## 8. 后续优先级建议

4.1~4.6 已全部完成，当前无遗留项；升级注意事项见第 7 节。
