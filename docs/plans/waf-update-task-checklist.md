# LogFlux WAF 更新管理任务清单（执行版）

> 对应设计文档：`docs/plans/waf-update-management-design.md`

## 1. 里程碑与范围

- 目标：实现 CRS 规则集的“源地址自动下载 + 手动上传 + 安全激活 + 自动回滚 + 定时检查”。
- 范围：后端（go-zero + GORM）为主，前端只做最小管理页面。
- 不含：在线升级 Coraza 二进制（仅做版本检查与升级建议）。

## 2. 任务优先级说明

- `P0`：必须上线（核心闭环，缺失即不可用）
- `P1`：建议上线（提高稳定性、可观测性）
- `P2`：增强项（后续优化）

## 3. 阶段任务清单

## 阶段 A：数据与配置基础（P0）

- [x] `P0-A01` 新增模型：`backend/model/waf_source.go`
- [x] `P0-A02` 新增模型：`backend/model/waf_release.go`
- [x] `P0-A03` 新增模型：`backend/model/waf_update_job.go`
- [x] `P0-A04` 在 `backend/internal/svc/service_context.go` 加入 AutoMigrate
- [x] `P0-A05` 增加配置结构：`backend/internal/config/config.go` 的 `WAF` 节
- [x] `P0-A06` 在 `backend/etc/config.yaml` 增加默认 WAF 配置
- [x] `P0-A07` 启动时确保目录存在：`/config/security/{tmp,packages,releases}`

**验收标准**
- [x] 新表自动创建成功
- [x] 配置加载无报错
- [x] 首次启动目录结构自动创建

## 阶段 B：核心领域服务（P0）

- [x] `P0-B01` 新建 `backend/internal/waf/fetcher.go`（HTTP/GitHub 拉取）
- [x] `P0-B02` 新建 `backend/internal/waf/verifier.go`（SHA256、大小、后缀白名单）
- [x] `P0-B03` 新建 `backend/internal/waf/extractor.go`（防 zip-slip、符号链接逃逸）
- [x] `P0-B04` 新建 `backend/internal/waf/store.go`（release 入库、路径管理）
- [x] `P0-B05` 新建 `backend/internal/waf/activator.go`（原子切换 current、失败回滚）
- [x] `P0-B06` 复用 Caddy `/adapt` + `/load` 做激活验证
- [x] `P0-B07` 增加并发锁（激活互斥，避免并发切换）

**验收标准**
- [x] `sync -> verify -> activate` 全链路可跑通
- [x] 激活失败可自动回退到 `last_good`
- [x] 并发激活仅允许一个任务执行

## 阶段 C：API 与 Handler/Logic（P0）

- [x] `P0-C01` 在 `backend/api/manage.api` 新增 WAF source/release/job/upload 接口
- [x] `P0-C02` 使用 `goctl api go -api api/logflux.api -dir . --style go_zero` 生成代码
- [x] `P0-C03` 实现 source CRUD logic
- [x] `P0-C04` 实现 `source/:id/check` 与 `source/:id/sync`
- [x] `P0-C05` 实现 `upload`（multipart）
- [x] `P0-C06` 实现 `release/:id/activate`
- [x] `P0-C07` 实现 `release/rollback`
- [x] `P0-C08` 实现 `job` 列表查询

**验收标准**
- [x] API 文档与返回结构符合项目统一 `code/msg/data`
- [x] 错误分支有可读错误信息（参数、校验、下载、回滚）

## 阶段 D：调度与任务系统（P1）

- [ ] `P1-D01` 新建 `backend/internal/tasks/waf_scheduler.go`
- [ ] `P1-D02` 从 `waf_sources.schedule` 动态装载任务
- [ ] `P1-D03` 支持启动加载、变更重载、手动触发
- [ ] `P1-D04` 任务执行写入 `waf_update_jobs`

**验收标准**
- [ ] 定时任务按 cron 正常触发
- [ ] 失败任务有完整审计记录

## 阶段 E：通知与告警（P1）

- [ ] `P1-E01` 在 `backend/internal/notification/event.go` 增加 WAF 事件常量
- [ ] `P1-E02` 在 check/sync/activate/rollback 成败处发通知
- [ ] `P1-E03` 增加默认事件订阅建议文档

**验收标准**
- [ ] 关键动作触发通知可追踪

## 阶段 F：前端最小能力（P1）

- [ ] `P1-F01` 新增“WAF 更新管理”菜单路由（仅 admin）
- [ ] `P1-F02` 源列表 + CRUD
- [ ] `P1-F03` 发布版本列表 + 激活 + 回滚
- [ ] `P1-F04` 上传规则包页面
- [ ] `P1-F05` 任务日志列表与状态过滤

**验收标准**
- [ ] 全流程可通过 UI 完成（无需手工调接口）

## 阶段 G：安全与稳定性（P0/P1）

- [x] `P0-G01` 下载域名白名单（默认仅 HTTPS）
- [x] `P0-G02` 上传大小限制与文件类型限制
- [x] `P0-G03` 包解压安全防护（zip-slip、symlink、文件数上限）
- [ ] `P1-G04` release 保留策略（仅保留最近 N 个）
- [ ] `P1-G05` 激活超时控制与重试策略

**验收标准**
- [ ] 安全测试通过（恶意包/异常路径/超大文件）

## 阶段 H：测试与发布（P0）

- [x] `P0-H01` 单测：verifier/extractor/activator
- [ ] `P0-H02` 单测：logic 层关键失败分支
- [ ] `P0-H03` 集成测试：模拟下载、校验、激活、回滚
- [ ] `P0-H04` 发布前演练：手动上传 + 激活失败自动回滚
- [ ] `P0-H05` 发布后验证：30 分钟核心指标观察

**验收标准**
- [ ] 自动化测试通过
- [ ] 演练记录与回滚记录完备

## 4. 接口落地顺序建议（开发节奏）

1. Source CRUD
2. 手动上传 + release 入库
3. activate + rollback
4. source check/sync
5. job 列表
6. scheduler 自动化

## 5. 工时预估（可并行）

- 后端核心（A+B+C+G+H）：约 6~9 人日
- 调度与通知（D+E）：约 2~3 人日
- 前端最小能力（F）：约 2~4 人日
- 合计：约 10~16 人日

## 6. 上线前 DoD 清单

- [ ] 可新增远程 CRS 源并完成一次同步
- [ ] 可上传规则包并激活
- [ ] 激活失败自动回滚可验证
- [ ] 任务日志可查询失败原因
- [ ] 安全限制全部生效（域名、大小、解压）
- [ ] 回归验证通过（登录、核心 API、配置保存）

## 7. 文档任务完成记录

- [x] `DOC-01` 总体设计文档：`waf-update-management-design.md`
- [x] `DOC-02` 执行任务清单：`waf-update-task-checklist.md`
- [x] `DOC-03` 运维操作指南：`waf-update-operations-guide.md`
- [x] `DOC-04` P0 实施细化：`waf-update-p0-implementation-plan.md`
