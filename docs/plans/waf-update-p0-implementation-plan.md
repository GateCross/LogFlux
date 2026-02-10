# LogFlux WAF 更新管理 P0 实施细化计划（开发可执行版）

> 关联文档：
> - 设计：`docs/plans/waf-update-management-design.md`
> - 任务清单：`docs/plans/waf-update-task-checklist.md`
> - 运维指南：`docs/plans/waf-update-operations-guide.md`

## 1. 目标

在不改 `docker-compose` 挂载的前提下，完成 WAF CRS 更新闭环（P0）：

- 源配置管理（source CRUD）
- 下载/上传规则包（sync/upload）
- 校验与安全解压（verify/extract）
- 激活与回滚（activate/rollback）
- 任务审计（jobs）

## 2. P0 交付范围

## 2.1 必须完成的 API

- `GET /api/caddy/waf/source`
- `POST /api/caddy/waf/source`
- `PUT /api/caddy/waf/source/:id`
- `DELETE /api/caddy/waf/source/:id`
- `POST /api/caddy/waf/source/:id/check`
- `POST /api/caddy/waf/source/:id/sync`
- `POST /api/caddy/waf/upload`
- `GET /api/caddy/waf/release`
- `POST /api/caddy/waf/release/:id/activate`
- `POST /api/caddy/waf/release/rollback`
- `GET /api/caddy/waf/job`

## 2.2 必须完成的文件落地

- 模型：
  - `backend/model/waf_source.go`
  - `backend/model/waf_release.go`
  - `backend/model/waf_update_job.go`
- 领域服务：
  - `backend/internal/waf/fetcher.go`
  - `backend/internal/waf/verifier.go`
  - `backend/internal/waf/extractor.go`
  - `backend/internal/waf/store.go`
  - `backend/internal/waf/activator.go`
- 接口层（goctl 生成后补 logic）：
  - `backend/internal/handler/caddy/*waf*.go`
  - `backend/internal/logic/caddy/*waf*.go`

## 3. 分步实施（建议顺序）

## Step 1：模型与配置底座

1. 新增三张表模型（source/release/job）。
2. 在 `backend/internal/svc/service_context.go` 增加 AutoMigrate。
3. 在 `backend/internal/config/config.go` 增加 `WAF` 配置节：
   - `WorkDir`
   - `MaxPackageBytes`
   - `AllowedDomains`
   - `ExtractMaxFiles`
   - `ExtractMaxTotalBytes`
   - `ActivateTimeoutSec`
4. 在 `backend/etc/config.yaml` 提供默认值。

**完成标准**
- 服务启动后自动建表成功。
- 未配置时有合理默认值。

## Step 2：规则包处理核心能力

1. `verifier.go`
   - 校验扩展名（`.tar.gz` / `.zip`）
   - 校验文件大小
   - 校验 SHA256（生产建议必填）
2. `extractor.go`
   - 解压前检查路径穿越
   - 禁止符号链接逃逸
   - 限制文件数量、总解压大小
3. `store.go`
   - 统一生成 release 路径
   - 维护 `current` / `last_good` 软链

**完成标准**
- 恶意包在解压前被拦截。
- 合法包可落盘为版本目录。

## Step 3：激活与回滚

1. `activator.go`
   - 获取互斥锁（DB/Redis 二选一）
   - 切换 `current` -> 新版本
   - 调用现有 Caddy `/adapt` + `/load`
   - 失败自动切回 `last_good`
2. `release` 状态机：
   - `verified -> active`
   - 失败标记 `failed`
   - 回滚目标标记 `rolled_back`

**完成标准**
- 激活失败时无需人工干预即可恢复旧版本。

## Step 4：API 落地（go-zero）

1. 修改 `backend/api/manage.api`。
2. 执行 goctl 生成：

```bash
cd backend
goctl api go -api api/logflux.api -dir . --style go_zero
```

3. 在 `logic/caddy` 实现业务逻辑。
4. 统一返回 `code/msg/data`（沿用 `common/result`）。

**完成标准**
- Postman 可完整走通 source -> sync/upload -> activate -> rollback。

## Step 5：任务审计与可追踪

1. 所有关键动作写入 `waf_update_jobs`：
   - `check/download/verify/activate/rollback`
2. 失败必须记录可读 `message`。
3. 提供 `job` 分页查询与状态过滤。

**完成标准**
- 失败问题可通过 jobs 快速定位。

## 4. 数据结构建议（最简）

## 4.1 `waf_sources`

- `name` 唯一索引
- `kind` 枚举：`crs|coraza_engine`
- `mode` 枚举：`remote|manual`
- `enabled`、`auto_check`、`auto_download`、`auto_activate`

## 4.2 `waf_releases`

- 组合索引：`source_id + version`
- `status` 枚举：`downloaded|verified|active|failed|rolled_back`
- `storage_path` 使用绝对路径（方便排障）

## 4.3 `waf_update_jobs`

- 索引：`source_id`、`status`、`created_at`
- `message` 存错误摘要（不存超大日志正文）

## 5. 安全基线（P0 强制）

- 仅允许 HTTPS 源地址。
- 域名白名单（例如 `github.com`, `api.github.com`）。
- 上传包大小上限（例如 100MB）。
- 解压文件数量与总大小限制。
- 禁止软链逃逸与路径穿越。

## 6. 测试清单（P0）

## 单测

- `verifier`：
  - 正常包
  - 错误后缀
  - 超大小
  - SHA 不匹配
- `extractor`：
  - 正常解压
  - zip-slip 攻击包
  - symlink 攻击包

## 集成测试

- 上传合法包 -> verified
- activate 成功 -> active
- activate 失败 -> 自动回滚

## 7. 发布前检查

- API 权限仅 `admin` 可操作。
- 审计日志与任务日志可查询。
- 回滚演练至少一次通过。

## 8. 预计排期（单人）

- Step1：0.5~1 天
- Step2：1~1.5 天
- Step3：1 天
- Step4：1~1.5 天
- Step5 + 测试：1 天

合计：约 4.5~6 天。

