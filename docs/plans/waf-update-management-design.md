# LogFlux WAF 规则更新管理设计（源地址下载 + 手动上传）

## 1. 背景与目标

你当前希望实现：
- Coraza/CRS 可定期更新；
- 不依赖 `docker-compose` 额外挂载目录；
- 支持“配置源地址自动下载”与“手动上传规则包”两种方式。

本设计基于当前仓库现状：
- 已有持久化卷 `caddy_config:/config/caddy`（可作为规则仓库存储目录）；
- 已有 Caddy 配置热加载链路 `/adapt` + `/load`；
- 已有 Caddy 配置历史回滚能力。

## 2. 关键事实（必须先统一）

### 2.1 CRS 与 Coraza 的更新性质不同
- **CRS（规则集）**：可通过下载/上传实现在线更新。
- **Coraza（Caddy WAF 模块）**：随 Caddy 二进制编译，不能仅靠下载规则热更新，必须重建镜像并发布。

### 2.2 本方案覆盖范围
- 重点实现 **CRS 生命周期管理**（check/download/verify/activate/rollback）。
- 对 Coraza 提供 **版本检查与升级建议能力**，实际升级仍走镜像发布流程。

## 3. 总体架构

```
Admin UI
  -> LogFlux Backend (WAF Update Manager)
       -> Source Fetcher (GitHub/HTTP)
       -> Upload Ingestor (multipart)
       -> Verifier + Extractor
       -> Release Store (/config/security/releases/...)
       -> Activator (switch current + /adapt + /load)
       -> Scheduler (periodic check/sync)
       -> Audit & Job History (DB)
  -> Caddy Admin API
```

## 4. 存储设计（不改 compose）

使用现有持久化路径 `/config/caddy`，新增目录结构：

```
/config/security/
  sources/                       # 源配置缓存（可选）
  packages/                      # 原始下载包/上传包
  releases/
    crs-v4.23.0/
      rules/
      crs-setup.conf
    crs-v4.23.1/
      ...
  current -> /config/security/releases/crs-v4.23.1   # 软链
  last_good -> /config/security/releases/crs-v4.23.0 # 软链
  tmp/
```

`docker/Caddyfile` 中固定引用：
- `Include /config/security/current/...`

这样无需新增 compose 挂载即可持久化规则与版本切换。

## 5. 数据模型设计（GORM）

## 5.1 `waf_sources`

用途：管理下载源配置（CRS/Coraza）。

建议字段：
- `id`
- `name`（唯一）
- `kind`：`crs` | `coraza_engine`
- `mode`：`remote` | `manual`
- `url`（release API 或包地址模板）
- `checksum_url`（可选）
- `auth_type`：`none` | `token` | `basic`
- `auth_secret`（加密存储）
- `schedule`（cron 表达式）
- `enabled`
- `auto_check`
- `auto_download`
- `auto_activate`
- `last_checked_at`
- `last_release`
- `last_error`
- `created_at` / `updated_at`

## 5.2 `waf_releases`

用途：记录已导入规则包版本。

建议字段：
- `id`
- `source_id`
- `kind`：`crs` | `coraza_engine`
- `version`
- `artifact_type`：`tar.gz` | `zip` | `upload`
- `checksum`
- `size_bytes`
- `storage_path`
- `status`：`downloaded` | `verified` | `active` | `failed` | `rolled_back`
- `meta`（jsonb，存扩展信息）
- `created_at` / `updated_at`

## 5.3 `waf_update_jobs`

用途：审计更新任务与故障排查。

建议字段：
- `id`
- `source_id`
- `release_id`
- `action`：`check` | `download` | `verify` | `activate` | `rollback`
- `trigger_mode`：`manual` | `schedule` | `upload`
- `operator`
- `status`：`running` | `success` | `failed`
- `message`
- `started_at` / `finished_at`
- `created_at` / `updated_at`

## 6. API 设计（基于 go-zero /api + caddy group）

建议在 `backend/api/manage.api` 的 `caddy` 分组下新增。

## 6.1 源配置管理
- `GET /caddy/waf/source`
- `POST /caddy/waf/source`
- `PUT /caddy/waf/source/:id`
- `DELETE /caddy/waf/source/:id`

## 6.2 源触发动作
- `POST /caddy/waf/source/:id/check`（仅检查新版本）
- `POST /caddy/waf/source/:id/sync`（下载+校验+可选激活）

## 6.3 手动上传
- `POST /caddy/waf/upload`
  - `multipart/form-data`
  - 字段建议：`file`、`kind`、`version`、`checksum`、`activateNow`

## 6.4 版本管理
- `GET /caddy/waf/release`
- `POST /caddy/waf/release/:id/activate`
- `POST /caddy/waf/release/rollback`

## 6.5 任务审计
- `GET /caddy/waf/job`

## 6.6 引擎版本检查（Coraza）
- `GET /caddy/waf/engine/status`
- `POST /caddy/waf/engine/check`

> 说明：`engine/check` 只做“发现新版本 + 生成升级建议”，不直接在线替换 Caddy 二进制。

## 7. 关键流程设计

## 7.1 远程同步流程（CRS）

1. 读取 `waf_sources` 配置。
2. 拉取远端版本元数据（GitHub Release API 或 URL 模板）。
3. 比较本地最新版本；无更新则结束。
4. 下载包到 `/config/security/tmp/`。
5. 校验大小、SHA256（建议必填）。
6. 安全解压到新目录 `releases/<version>`：
   - 禁止路径穿越（zip slip）
   - 禁止符号链接逃逸
   - 限制单文件与总文件数
7. 写入 `waf_releases`（`verified`）。
8. 若 `auto_activate=true`：执行激活流程。

## 7.2 手动上传流程

1. Handler 接收 `multipart` 文件。
2. 临时落盘并执行相同校验与解压流程。
3. 写入 `waf_releases`。
4. `activateNow=true` 时触发激活；否则仅入库待激活。

## 7.3 激活流程（核心）

1. 获取全局互斥锁（DB 锁或 Redis 锁，避免并发激活）。
2. 记录当前 `current` 目标到 `last_good`。
3. 原子切换 `current` 软链到目标版本目录。
4. 使用现有 Caddy Server 配置执行：
   - `/adapt` 预校验
   - `/load` 正式加载
5. 成功：更新 `waf_releases.status=active`。
6. 失败：恢复 `current` 到 `last_good`，并再次 `/load` 回退配置。

## 7.4 回滚流程

1. 选择 `last_good` 或指定历史 release。
2. 软链切换 + `/load`。
3. 记录 `waf_update_jobs(action=rollback)` 与审计日志。

## 8. 调度设计（定期更新）

新增专用调度器 `WAFScheduler`（不要复用 shell 脚本 CronTask）：
- 使用 `robfig/cron`，从 `waf_sources.schedule` 动态装载任务。
- 执行动作为 `check` 或 `sync`。
- 支持启停、重载、手动触发。

接入位置建议：
- `backend/internal/tasks/waf_scheduler.go`
- 在 `backend/internal/svc/service_context.go` 初始化并启动。

## 9. Coraza 更新策略（与 CRS 区分）

Coraza 在线热更新不可行，建议流程：

1. `engine/check` 检查上游模块版本。
2. 生成“可升级”记录并发通知（事件：`waf.engine_update_available`）。
3. 由 CI/CD 执行镜像升级：
   - 更新 `docker/caddy.Dockerfile` 的模块版本 pin
   - 触发 `.github/workflows/build-caddy.yml`
   - 灰度发布后全量。

## 10. 安全控制要求

- 仅允许 `https` 下载源（生产默认禁止 `http`）。
- 对下载域名做白名单（如 `github.com`, `api.github.com`）。
- 强制包体大小上限（如 100MB）。
- 强制 SHA256 校验（生产环境建议必选）。
- 上传文件类型白名单（`.tar.gz` / `.zip`）。
- 清理策略：仅保留最近 N 个 release（例如 5 个）。

## 11. 代码落地清单（建议顺序）

1. `backend/model/` 新增：
   - `waf_source.go`
   - `waf_release.go`
   - `waf_update_job.go`
2. `backend/internal/svc/service_context.go`
   - AutoMigrate 新表
   - 初始化 `WAFScheduler`
3. `backend/internal/config/config.go` + `backend/etc/config.yaml`
   - 增加 `WAF` 配置（工作目录、超时、大小限制、白名单）
4. `backend/api/manage.api`
   - 新增 WAF endpoints 与 types
5. 运行 goctl 生成代码（保持 `--style go_zero`）
6. `backend/internal/handler/caddy/` + `backend/internal/logic/caddy/`
   - 实现 source/release/job/upload/activate/rollback 逻辑
7. `backend/internal/waf/`
   - fetcher、verifier、extractor、activator
8. `docker/Caddyfile`
   - 固定 include 到 `/config/security/current`

## 12. 验收标准（DoD）

- 可配置至少一个远程 CRS 源，并可手动触发同步。
- 可通过上传规则包创建 release，并支持激活。
- 激活失败时能自动回退到 `last_good`。
- 可查看完整 job 历史与失败原因。
- 可定时检查更新并稳定运行一周。
- Coraza 版本更新可被发现并形成升级建议记录。

---

## 附录：建议新增通知事件

- `waf.source_check_failed`
- `waf.release_downloaded`
- `waf.release_verify_failed`
- `waf.release_activated`
- `waf.release_rollback`
- `waf.engine_update_available`

