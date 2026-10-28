# Caddy 配置架构

> 更新：2026-10-28
> 范围：`backend/internal/logic/caddy/`、`backend/model/caddy/`、`frontend/apps/src/views/caddy/`

Caddy 是 LogFlux 的唯一反向代理与 WAF 载体。本文记录**当前实现**的架构约束、站点模型、日志链路与尚未落地的技术债。

---

## 1. 单一真相源：Caddyfile 文本

系统的模型是「**文本权威 + 单一写入器**」，不是「结构化模型权威」。

| 事实 | 说明 |
|---|---|
| 配置唯一存储 | `system_configs` 表的 `caddy_config` 键，值为 `{config, modules}` |
| `config` | Caddyfile 全文，**唯一权威** |
| `modules` | 结构化配置 JSON，仅用于前端展示与快速编辑，不参与渲染 |
| 写入唯一入口 | `caddyConfigApplyService.Apply`，全局互斥锁 `caddyConfigWriteMu` |

写入链路固定为：

```
adapt（校验）→ load（热加载）→ persist（落库）
```

三者必须作为整体执行。任一步失败都回滚运行态，不落库。全局单锁的存在理由：无锁时两个并发写会交错执行（A.load → B.load → B.persist → A.persist），导致运行态与库内配置分叉且双方都返回成功。

### 1.1 保真原则

站点块按「定点替换」处理，不做整体重渲染：

- 改上游、改 TLS、开关防护能力，都只替换目标行/目标块，其余行**字节级不变**
- 无法识别的指令、`import` 片段、自定义块一律原样保留（`preservedBlocks`）
- 复杂站点（含无法归类 `import`）在 UI 里原文只读，但防护开关仍可调

这条原则的来源是一次真实故障：早期实现按结构化模型重新渲染整份 Caddyfile，导致用户手写的 snippet 与格式化细节被覆盖。**任何新增能力都必须走定点替换，不要引入第二条渲染路径。**

### 1.2 其他单例配置键

`system_configs` 里与 Caddy 相关的键：

| 键 | 内容 |
|---|---|
| `caddy_config` | Caddyfile 全文 + modules |
| `caddy_entry_block` | `:80` 系统入口模板 |
| `caddy_global_options` | 全局块选项（`order coraza_waf first` 等） |
| `caddy_log_snippet` | 站点日志 snippet 定义 |
| `caddy_log_retention` | 日志保留策略 `{enabled, days}`，缺省 7 天 |
| `caddy_config_migrated` | 单实例改造的配置迁移完成标记 |

---

## 2. 单实例模型：没有「节点」

**LogFlux 永远只有一个 Caddy 实例，产品里不存在「节点」概念。**

| 项 | 结论 |
|---|---|
| `caddy_servers` 表 | 仅作为迁移来源保留，不再读写；新部署为空表 |
| `serverId` 参数 | 全链路已移除（后端约 70 处、前端约 195 处） |
| remote 远端节点 | 能力已删除，不是隐藏而是不存在 |
| 多节点数据 | 启动时若检测到 `caddy_servers` 行数 > 1，只保留 id 最小的配置并告警，不自动删除用户数据 |

配置迁移顺序（已由 `service_context.go` 自动执行，幂等）：

1. `system_configs` 已有 `caddy_config` → 跳过
2. `caddy_servers` 不存在或为空 → 跳过
3. 取 id 最小一行，把 `Config`/`Modules` 写入 `caddy_config`
4. 写入迁移标记 `caddy_config_migrated`

**不要重新引入节点选择器、在线状态探测或 `serverId` 路径参数。** 单实例下这些控件恒为同一个值，只会让用户困惑「我该配几个节点」。

---

## 3. 站点模型：站点是唯一一等对象

产品入口是「站点」，不是「Caddyfile 文本」。一次建站 = 域名 + 上游地址，两步完成。

### 3.1 能力模型

站点的每一项能力都从**站点块文本识别**，三态输出：

| 状态 | 含义 | UI 行为 |
|---|---|---|
| `on` | 已启用 | 开关选中 |
| `off` | 未启用 | 开关未选中 |
| `unknown` | 文本里有相关内容但认不出来 | 开关禁用，引导去原文模式 |

识别的能力（`site-capabilities.ts`）：

| 能力 | 对应 snippet / 内联痕迹 |
|---|---|
| `waf` | `import waf_protect`；内联 `coraza_waf` 视为 unknown |
| `geo` | `import geo_auth`；内联 `forward_auth localhost:8888` |
| `compression` | `import compression`；内联 `encode` |
| `securityHeaders` | `import security_headers`；内联 `header` 视为 unknown |
| `accessLog` | `import subdomain-log`；内联 `log {` |
| `errorPage` | `import deny_403_redirect`；内联 `handle_errors` 视为 unknown |
| `tls` | TLS 模式：`auto` / `dns` / `internal` / `manual` / `off` |

`cloudflare` 属于已归类 import（DNS-01 证据），不需要开关控制。

`otherImports` 保留所有未归类到能力的站点级 `import`（`health_check`、业务片段等），保证任何改动都不丢用户的配置。

### 3.2 防护默认统一

新建站点**不问任何防护问题**：

- 全局 WAF 已启用 → 保存后自动挂载 `import waf_protect`
- 全局 WAF 未启用 → 不挂载

站点级差异（WAF 开关、IP 限制、安全头、错误页）在站点详情的防护 Tab 里按站点覆盖。强度、运行模式等参数**全局只有一份**，不做站点级策略对象。

### 3.3 前端页面结构

| 路由 | 页面 | 菜单 |
|---|---|---|
| `/caddy/catalog` | 站点（主入口，卡片网格） | 显示 |
| `/caddy/catalog/site` | 站点详情（整页，`?host=`） | 隐藏 |
| `/caddy/certificates` | 证书中心 | 显示 |
| `/caddy/access` | 访问控制 | 显示 |
| `/caddy/settings` | 全局设置 | 显示 |
| `/caddy/config` | 配置管理（高级） | 隐藏，从全局设置进入 |

旧 `/caddy` 深链重定向到 `/caddy/catalog`。

---

## 4. 证书中心

双源合并，同一个库存表展示两种来源的证书。

| 来源 | 来源目录/接口 | 状态语义 |
|---|---|---|
| Caddy 自动签发 | Caddy 存储卷（`Caddy.CertStorageDir`，缺省 `/data/caddy/certificates`） | 扫描所得，随 Caddy 状态变化 |
| 用户上传 | `/config/certs/<域名>/` | 落库记录，可删除 |

- 每日 03:40 由 `CertSyncScheduler` 同步一次（`0 40 3 * * *`），也可在页面手动刷新
- 到期告警走通知事件 `cert.expiring`（D-14/D-7 warning、D-1 critical）与 `cert.expired`
- 证书缺口视图 = 「Caddyfile 里声明了 TLS 的站点」− 「库存里有的证书」
- Cloudflare DNS-01 签发通过 `{$CF_API_TOKEN}` 环境变量注入，token 不落库

---

## 5. 站点日志

### 5.1 Caddy 能力边界（不要推翻这四条）

这部分是查证 Caddy 官方行为得到的结论，后续改动必须先读：

**① `level` 是最低级别，不是白名单。** 站点块内的 `log` 只有一个 `level`：

| 访问日志 | 错误日志 | 可写的 `level` | 可行 |
|---|---|---|---|
| 开 | 开 | `INFO` | ✅ |
| 关 | 开 | `ERROR` | ✅ |
| 关 | 关 | 不写 `log` | ✅ |
| 开 | 关 | — | ❌ **无法表达** |

UI 对「访问开 + 错误关」禁用并给出说明。

**② 错误日志无法按站点隔离命名空间。** 站点块内的 `log` 只控制访问日志；错误日志的 logger 命名空间是模块级（`http.log`、`tls` 等），不存在「某站点的错误日志」独立命名空间。**「访问/错误分开」只能在采集与查询层做**：Caddy 用 `level INFO` 写同一文件，LogFlux 采集时按 JSON 里的 `level` / `logger` 字段分流。

**③ `roll_size` 是滚动归档，不是截断覆盖。** 写满后改名为 `<name>-<时间戳>-size.log` 再新建文件。精度 1MB（`1.5MB` 向上取整为 `2MB`）；`roll_keep 0` 被 Caddy 当成未设置，回落默认 10。Caddy **没有**「超限清空重写」模式——真正单文件只能由后端轮询清空，会与 Caddy 写入竞争并丢日志，因此不采用。

**④ 归档文件必须排除。** 采集侧用正则排除 `-size` / `-time` 后缀（含 `.gz`），并要求完整时间戳 `-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}(?:\.\d+)?-(?:size|time)\.log(?:\.gz)?$`。曾因正则过宽把 `file-size.log` 误判为归档导致真实日志漏采，现有单测锁定。

### 5.2 采集链路

| 机制 | 实现 |
|---|---|
| 目录源注册 | `EnsureSiteLogSource` 把 `/var/log/caddy` 注册为目录源，新站点文件自动发现 |
| 启动加载 | 启动即注册目录源并加载所有 `type='caddy'` 的启用源 |
| 文件重建检测 | `LogIngestCursor.FileID` 记录 inode（dev+ino），文件重建后从 0 重采而非错位续读 |
| 字段分流 | `caddy_logs` 的 `level` / `logger` 列；JSON 日志读实际值，Coraza 审计与 console 文本固定 `info` / `http.log.access` |

`log_sources` 仍保留 `server_id` 列（恒为 0），唯一索引为 `(path, server_id)`。升级旧部署时若曾存在跨节点同路径数据，需先检查：

```sql
SELECT path, COUNT(*) FROM log_sources GROUP BY path HAVING COUNT(*) > 1;
```

### 5.3 保留策略

以 `system_configs` 的 `caddy_log_retention` 为唯一真值，缺省 `{enabled: true, days: 7}`。

| 项 | 行为 |
|---|---|
| 缺省回落 | 键不存在、JSON 非法、days 越界（<1 或 >3650）按 7 天 |
| 关闭 | `enabled=false` 时清理任务跳过本轮，查询接口照常返回配置 |
| 删除 | 每天 02:00 删除 `log_time` 早于「现在 − days」的行；单批 2000 行、单轮最多 500 批 |
| 读库失败 | **跳过本轮并告警，不回落默认天数**（`LoadLogRetentionStrict`），与查询接口的宽容读取刻意不同 |

归档机制已整体移除（归档表、存储过程、`CaddyLogArchive` 结构体）。数据库里的旧对象不会自动删除，需要时手工执行：

```sql
DROP TABLE IF EXISTS caddy_logs_archive;
DROP FUNCTION IF EXISTS archive_old_logs(INTEGER);
```

---

## 6. 尚未落地的技术债

以下三项已在设计文档中论证但**当前代码里没有**，不是「已完成」：

### 6.1 单一文本变换层未抽取

现状：`formatCaddyfile`、`countCaddyLineBraces`、`stripCaddyComment` 等文本手术函数仍在 `internal/logic/caddy/caddy_config_service.go` 内，与 svc/model 依赖混在同一包。

目标：抽到独立包（如 `internal/caddyfile`），使文本变换不依赖 `model` / `svc`，站点详情抽屉与全局设置页可共用同一套手术函数。

### 6.2 反引号块内的花括号格式化缺陷未修

`formatCaddyfile` 按行统计花括号缩进，但对反引号块（Coraza `directives` 里的 `SecMarker {`）不敏感，会把块内内容误当结构缩进。

目标：反引号块内内容逐行保持不变；保留现有幂等性断言（连续调用两次，第二次 `changed == false`）。

### 6.3 后端 render 双模型未删

`renderCaddyFormModel`（约 360 行）仍存在，唯一活调用点是 `caddy_config_service.go` 的空 config 分支：

```go
caddyfile := config
if strings.TrimSpace(caddyfile) == "" {
    caddyfile = renderCaddyFormModel(model)   // 前端总是传非空，不可达
}
```

前端确实从不传空 config，但直接调 API 的调用方可触发。删除 = 变更 API 行为，需要单独决策。

### 6.4 配置历史无 retention

`caddy_config_history` 无上限清理，长期高频保存会无限增长。设计目标为每实例保留最近 100 条，当前未实现。

---

## 7. 配置历史与回滚

| 能力 | 接口 |
|---|---|
| 列表 | `GET /caddy/config/history` |
| 详情 | `GET /caddy/config/history/:historyId` |
| 回滚 | `POST /caddy/config/rollback` |

- 每次 apply 落一条历史，含 `hash`（当前配置哈希）与 `base_hash`（写入前哈希）
- `baseHash` 用于乐观并发：客户端提交时带上次读到的 hash，不一致则拒绝写入（「配置已被他人修改，请刷新后重试」）。`baseHash` 为空视为旧客户端，直接放行
- 回滚失败时需同时回退运行态与库内配置，两侧都要有测试锁定

---

## 8. 相关接口一览

| 域 | 接口 |
|---|---|
| 配置 | `GET/POST /caddy/config`、`POST /caddy/config/preview`、`GET /caddy/config/history`、`POST /caddy/config/rollback` |
| 全局 | `GET/PUT /caddy/global-options`、`GET/PUT /caddy/entry-block`、`POST /caddy/entry-block/reset-default`、`GET/PUT /caddy/log-snippet`、`POST /caddy/log-snippet/reset-default` |
| 日志 | `GET /caddy/logs`、`POST /caddy/logs/clear`、`POST /caddy/logs/site-metrics`、`GET/PUT /caddy/log-retention` |
| 证书 | `GET /caddy/certificates`、`POST /caddy/certificates/upload`、`POST /caddy/certificates/refresh`、`DELETE /caddy/certificates/:id` |
| 访问控制 | `GET/PUT /caddy/ip-region` |
| 发现 | `GET /caddy/discovery/docker` |

WAF 相关接口见 [`waf.md`](./waf.md)。
