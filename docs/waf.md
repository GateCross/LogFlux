# WAF（Coraza / CRS）

> 更新：2026-10-28
> 范围：`backend/api/waf.api`、`backend/internal/logic/caddy/`（WAF 部分）、`backend/internal/waf/`、`frontend/apps/src/views/caddy/settings/`

LogFlux 的 WAF 由 **Coraza + OWASP CRS** 组成，编译进定制 Caddy 镜像。本文覆盖运行时策略、规则更新与运维处置。

> **历史说明**：早期版本有独立的「安全管理」页面，包含策略版本、规则例外、策略绑定、误报反馈、发布/回滚等能力。该页面与对应接口已在 2026-06 整体删除（提交 `af00eaf`）。**`/security` 路由、`manage.api`、`WafRuleExclusion` / `WafPolicyBinding` / `WafPolicyFalsePositiveFeedback` 模型、`/caddy/waf/release*` 与 `/caddy/waf/job` 接口都已不存在**，不要按旧文档实施。

---

## 1. 能力边界：CRS 可热更新，Coraza 不可

| 组件 | 可否在线更新 | 说明 |
|---|---|---|
| **CRS（规则集）** | ✅ 可 | 下载/上传 → 校验 → 重构布局 → 激活 → 热加载 |
| **Coraza（WAF 模块）** | ❌ 不可 | 随 Caddy 二进制编译，升级必须重建镜像并发布 |

因此 Coraza 只提供**版本检查与升级建议**，实际升级走 `docker/caddy.Dockerfile` 的版本 pin + `.github/workflows/build-caddy.yml`。

---

## 2. 运行时策略（simple-config）

### 2.1 三个维度

| 维度 | 取值 | 作用 |
|---|---|---|
| 运行模式 `mode` | `off` / `detectiononly` / `on` | 关闭 / 仅检测 / 阻断 |
| 强度 `strength` | `low_fp` / `balanced` / `high_blocking` | 异常阈值 + PL 档位 |
| 审计 `audit` | `off` / `relevantonly` / `on` | `SecAuditEngine` 级别 |

> 不要把 `mode` 与 `enabled` 混为一谈：`enabled` 决定 WAF 是否**挂载到站点**（Caddyfile 里有没有 `import waf_protect`），`mode` 决定挂载后**怎么跑**。`enabled=false` 时整块 `waf_protect` 不出现。

### 2.2 强度映射

强度通过两条 `SecAction` 生效，PL 必须用 CRS 4.x 的变量名：

```
SecAction "id:900000,phase:1,pass,nolog,t:none,setvar:tx.blocking_paranoia_level=<PL>"
SecAction "id:900001,phase:1,pass,nolog,t:none,setvar:tx.detection_paranoia_level=<PL>"
SecAction "id:900110,phase:1,pass,nolog,t:none,setvar:tx.inbound_anomaly_score_threshold=<in>,setvar:tx.outbound_anomaly_score_threshold=<out>"
```

**`tx.paranoia_level` 是失效写法。** CRS 4.x 只读 `blocking_paranoia_level` / `detection_paranoia_level`；写裸 `paranoia_level` 会让 PL 恒等于 1，规则集永远停在最低档。`waf_policy_builder_test.go` 里有断言锁定这一点——出现裸 `tx.paranoia_level` 即视为回退。

约束：`detection_paranoia_level` 必须 ≥ `blocking_paranoia_level`，否则 CRS 规则 901500 直接报错。

### 2.3 请求体限制

| 字段 | 对应指令 |
|---|---|
| `requestBodyAccess` | `SecRequestBodyAccess On/Off` |
| `requestBodyLimit` | `SecRequestBodyLimit` |
| `requestBodyNoFilesLimit` | `SecRequestBodyNoFilesLimit` |

### 2.4 两条已修复的绕过缺陷（不要引入回去）

| 缺陷 | 后果 | 现状 |
|---|---|---|
| Host 头以 IP 开头的请求注入 `ctl:ruleEngine=Off` | 默认 `:80` 是 catch-all，任意客户端构造 `Host: 1.2.3.4` 即可跳过全部规则 | 已删除该规则 |
| `/api/` 全路径注入 `ctl:ruleEngine=Off` | 管理面（含登录）完全无 WAF 保护，且 raw 模式删不掉（每次保存被重新注入） | 已改为端点级 `ctl:requestBodyAccess=Off`，只关请求体检查，不关规则引擎 |

### 2.5 站点绑定

`siteAddresses` 保存在默认策略的 `Config` 里（键 `siteAddresses`），**不新建绑定表**。站点是否真正受保护由 Caddyfile 的 `import waf_protect` 决定，库里只记录用户意图，刷新页面不丢。

`siteAddresses` 为空时按「全部可识别站点」处理。

---

## 3. 规则仓库布局

工作目录固定 `/config/security`（`Waf.WorkDir` 运行时收敛到此，**不要改到别处**）。

```
/config/security/
  packages/                 # 原始下载包 / 上传包（含 .stage_* 临时文件）
  releases/
    <version>/              # 解压后的原始规则包
  active/                   # 当前生效的内嵌同构布局（Caddyfile 从这里加载）
    rules/
      @coraza.conf-recommended
      @crs-setup.conf.example
      @owasp_crs/*.conf
  current     -> releases/<version>      # 软链
  last_good   -> releases/<version>      # 软链
```

`active/` 是**生成物**：把 `releases/<version>` 重构成 coraza-caddy `load_owasp_crs` 期望的目录形状。绝对路径从 OSFS 读取，从而接管内嵌规则，让在线更新真正生效。

`docker/Caddyfile` 的固定引用：

```
Include /config/security/active/rules/@coraza.conf-recommended
Include /config/security/active/rules/@crs-setup.conf.example
Include /config/security/active/rules/@owasp_crs/*.conf
```

全新部署时 `active/` 不存在，Caddy 会因找不到 include 目标启动失败——`entrypoint.sh` 负责在首次启动时从内嵌规则生成布局兜底。

---

## 4. 规则更新接口

当前 WAF 相关接口共 **15 个端点**，全部挂在 `caddy` 分组、前缀 `/api`：

### 4.1 源管理

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/caddy/waf/source` | 源列表 |
| POST | `/caddy/waf/source` | 新增源 |
| PUT | `/caddy/waf/source/:id` | 更新源 |
| DELETE | `/caddy/waf/source/:id` | 删除源（连带其 job 记录） |
| POST | `/caddy/waf/source/:id/check` | 只检查新版本，不下载 |
| POST | `/caddy/waf/source/:id/sync` | 下载 + 校验 + 可选激活 |

### 4.2 上传与引擎

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/caddy/waf/upload` | `multipart/form-data`：`file`、`kind`、`version`、`checksum`、`activateNow` |
| GET | `/caddy/waf/engine/status` | Coraza 当前版本 / 最新版本 / 是否可升级 |
| POST | `/caddy/waf/engine/check` | 触发一次版本检查（只生成建议，不替换二进制） |

### 4.3 策略与事件

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/caddy/waf/simple-config` | 读当前策略 |
| PUT | `/caddy/waf/simple-config` | 写策略（只落库，不生效） |
| POST | `/caddy/waf/simple-config/preview` | 预览生成结果（不落库、不加载） |
| POST | `/caddy/waf/simple-config/apply` | 生成 → adapt → load → 落库 |
| GET | `/caddy/waf/events` | 拦截/检测事件查询（分页、时间、IP、Host、规则 ID、级别、动作、关键词） |
| GET | `/caddy/waf/events/stats` | 事件聚合统计 |

**`PUT` 与 `apply` 的区别很重要**：`PUT` 只改库；`apply` 才改运行态。前端保存按钮走 `apply`（带 diff 确认），草稿态走 `PUT`。

### 4.4 没有查询接口的数据表

`waf_releases` 与 `waf_update_jobs` 仍在写入，但**当前没有任何查询接口**，前端也零展示：

| 表 | 谁写 | 可见性 |
|---|---|---|
| `waf_releases` | 同步、上传、激活时建档，按 `Waf.ReleaseRetentionCount` 清理 | 无接口，纯内部 |
| `waf_update_jobs` | 每次 check/sync/activate 记审计；`engine_check` 记录另供引擎版本状态读取 | 仅 `engine/status` 间接用一条 |

排查更新问题时，这两张表需要直接查库：

```sql
-- 最近 20 条更新任务
SELECT id, source_id, release_id, action, trigger_mode, status, message, finished_at
FROM waf_update_jobs ORDER BY id DESC LIMIT 20;

-- 当前保留的 release
SELECT id, source_id, version, status, created_at
FROM waf_releases ORDER BY id DESC;
```

> 早期版本有 `/caddy/waf/release` 与 `/caddy/waf/job` 查询接口，已随 `/security` 页面删除。表从「可查询的运维数据」降级为「内部审计日志」。

---

## 5. 更新流程

### 5.1 远程同步

1. 读源配置 → 拉版本元数据（GitHub Release API 或 URL 模板）
2. 无更新则结束
3. 下载到 `packages/.stage_<name>`，完成后原子改名
4. 校验大小、SHA256
5. 安全解压到 `releases/<version>`：禁止路径穿越（zip-slip）、禁止符号链接逃逸、限制文件数与总字节
6. 重构 `active/` 布局（`BuildCRSLayout`）
7. `activateNow=true` 时执行激活

### 5.2 激活与自动回滚（核心）

```
1. 取全局互斥锁（避免并发激活）
2. 记录 current 指向到 last_good
3. 原子切换 current → 目标版本
4. /adapt 预校验 → /load 正式加载
5. 成功：release 状态置 active
6. 失败：current 恢复到 last_good，再次 /load 回退运行态
```

失败要留下 `WafUpdateJob` 记录（`action=activate`、`status=failed`、`message` 为中文原因），这是排障的第一入口。

### 5.3 安全控制

| 项 | 约束 |
|---|---|
| 协议 | 生产默认只允许 `https` |
| 域名白名单 | `Waf.AllowedDomains`，缺省 `github.com`、`api.github.com` |
| 包体上限 | `Waf.MaxPackageBytes`，缺省 100MB |
| SHA256 | 建议必填；不匹配立即终止，不激活 |
| 解压限制 | `ExtractMaxFiles`（5000）、`ExtractMaxTotalBytes`（512MB） |
| 保留数量 | `ReleaseRetentionCount`（缺省 20），清理历史 release 时**保留 active 版本** |
| 上传类型 | 只允许 `.tar.gz` / `.zip` |

---

## 6. 调度

| 任务 | 实现 | 频率 |
|---|---|---|
| WAF 源检查/同步 | `WafScheduler`，按 `waf_sources.schedule` 动态装载（`robfig/cron`） | 每源独立 cron |
| 证书同步 | `CertSyncScheduler` | 每日 03:40 |
| 日志清理 | `LogCleanupTask` | 每日 02:00 |

`WafScheduler` 是独立调度器，**不要用 shell 脚本 CronTask 替代**——它需要动态重载、启停与手动触发。

---

## 7. 通知事件

已实现的事件（`internal/notification/event.go`）：

| 事件 | 级别 |
|---|---|
| `security.waf_source_check_success` | info |
| `security.waf_source_check_failed` | error |
| `security.waf_source_sync_success` | info |
| `security.waf_source_sync_failed` | error |
| `security.waf_release_activate_success` | info |
| `security.waf_release_activate_failed` | error |

**建议至少订阅两条失败事件**：`security.waf_source_sync_failed`、`security.waf_release_activate_failed`。

> 早期文档里提到的 `security.waf_release_rollback_*` 不存在——回滚当前以 activate 失败时的错误消息体现，没有独立事件。

---

## 8. 运维处置

### 8.1 日常巡检（容器内）

```bash
# current / last_good 指向
ls -l /config/security/current /config/security/last_good

# 已安装版本
ls -lah /config/security/releases

# 生效布局是否完整（三个 include 目标都要在）
ls /config/security/active/rules/

# 审计日志
tail -n 200 /var/log/caddy/waf_audit.log

# 运行态日志
tail -n 200 /var/log/caddy/runtime.log
```

每日检查：最近 24h 是否有失败任务、`current` 是否与预期版本一致、`waf_audit.log` 是否持续写入、`/config/security` 磁盘占用。
每周检查：release 保留数量、源凭证是否过期、是否有长期未处理的失败记录。

### 8.2 失败场景

| 现象 | 常见原因 | 处置 |
|---|---|---|
| 下载失败 | 网络不通、域名不在白名单、token 失效 | 查 job `message`；校验源配置与凭证；必要时切手动上传 |
| 校验失败 | 包被篡改、checksum 配错 | 终止激活；重新取官方包与 checksum |
| 解压失败 | zip-slip、非法符号链接、超限 | 用可信来源重新打包；**不要强制绕过** |
| 激活失败 | `/adapt` 失败、include 目标缺失、规则语法错 | 已自动回滚 `last_good`；查 Caddy admin 返回与 job 日志 |
| 规则生效但 PL 未变 | 回退到了裸 `tx.paranoia_level` 写法 | 检查 `waf_policy_builder.go` 生成的 directives |

### 8.3 紧急更新（高危漏洞）

1. 立即 `sync`（可带 `activateNow=true`）
2. 验证拦截命中与关键业务可用性
3. 误报高则回滚到 `last_good`，再考虑排除规则
4. 复盘并更新值班手册

### 8.4 离线 / 内网更新

1. 离线环境下载规则包并生成 SHA256
2. 通过 `/caddy/waf/upload` 提交
3. 系统自动 `/adapt` 校验
4. 激活并验证

---

## 9. 代码落点

| 关注点 | 位置 |
|---|---|
| 策略 → directives 文本 | `internal/logic/caddy/waf_policy_builder.go` |
| simple-config 读写与 apply | `internal/logic/caddy/simple_waf_config_service.go` |
| 源/发布/任务辅助 | `internal/logic/caddy/waf_logic_helpers.go` |
| 通知映射 | `internal/logic/caddy/waf_update_notification_helpers.go` |
| 保留数量清理 | `internal/logic/caddy/waf_release_retention_helpers.go` |
| 抓取 / 校验 / 解压 / 激活 / 存储 | `internal/waf/{fetcher,verifier,extractor,activator,store}.go` |
| CRS 布局重构 | `internal/waf/crs_layout.go` |
| 调度 | `internal/tasks/waf_scheduler.go` |
| 前端入口 | `views/caddy/settings/{SettingsWafCard,WafSourcesCard}.vue` |
| 前端事件页 | `views/logs/security/index.vue` |
