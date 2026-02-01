# Notification 异步队列 + 4 Worker 并发池（设计草案）

## 背景与目标
当前通知系统在 `Notify` 内并发发送但同步等待完成（`backend/internal/notification/manager.go:166-228`），会把业务调用时延与外部依赖抖动耦合；同时存在多处并发安全与可靠性隐患（并发 map 写、Redis nil、SilenceDuration 静默期失效、日志状态码前后端不一致）。

本设计的目标是：
- **接口返回即成功**：`Notify` 只负责“入队”，不做网络发送。
- **最终可达**：后台异步派发，失败自动重试（按渠道自定义策略，默认开启）。
- **小型并发池**：单实例固定 4 个 worker 并发发送。
- **可观测**：UI 仍通过 `notification_logs` 展示发送结果与失败原因。

## 约束与取舍
- 部署：**单实例**（暂不考虑多实例抢占与分布式锁）。
- 重试：**按渠道自定义**；未配置则使用默认指数退避（方案 A）。
- 渠道配置/模板：**执行时读取最新配置并渲染模板**（避免因修复配置而无法恢复积压任务）。
- 语义：**至少一次（At-least-once）**。极端情况下可能重复发送（例如发送成功但落库前崩溃）。

## 现状问题（必须修复）
### 1) 并发写 map（可能 panic）
- `Notify` 并发发送多个渠道（`manager.go:207-228`），但 `sendToChannel` 会写入共享 `event.Data["rendered_content"]`（`manager.go:342-353`），存在 `concurrent map writes` 风险。
- 规则引擎缓存使用普通 `map`：
  - `ThresholdEvaluator.cache`（`rule_engine.go:93-128`）
  - `PatternEvaluator.cache`（`rule_engine.go:205-246`）
  并发触发下同样有数据竞争风险。

### 2) Redis 可选但 frequency 规则可能 nil pointer
- Redis 连接失败仍继续启动，`rdb` 可能为 nil（`service_context.go:55-66`）。
- frequency evaluator 直接 `f.redis.Incr(...)`（`rule_engine.go:189-193`）。

### 3) SilenceDuration 静默期失效导致洪泛
- 静默判断依赖内存 `rule.LastTriggeredAt`（`rule_engine.go:63-69`）。
- 触发后仅更新 DB，不更新内存 rule（`manager.go:256-269`）。

### 4) 通知日志 status 码前后端不一致
- 后端 `pending -> 1`（`get_notification_logs_logic.go:61-70`），前端 `1 -> sending`（`frontend/src/views/notification/log/index.vue:110-115`），且后端 `req.Status==0` 被当成不筛选（`get_notification_logs_logic.go:32-40`）。

## 目标架构
### 数据表
- **notification_jobs（新增）**：异步派发任务表（队列真源）。
- **notification_logs（保留）**：发送历史与展示表（UI 查询、审计）。

建议字段（jobs）：
- `id`
- `log_id`（关联 notification_logs）
- `channel_id`、`provider_type`
- `event_type`、`event_level`、`event_title`、`event_message`、`event_data(jsonb)`
- `template_name`（可选）
- `status`：queued / processing / succeeded / failed（或 queued/processing + 终态由 log 表体现）
- `retry_count`
- `next_run_at`
- `last_error`
- `created_at`、`updated_at`、`last_attempt_at`

建议字段（logs，补齐/统一）：
- `status`：pending / sending / success / failed（与前端一致的枚举含义）
- `error_message`、`sent_at`
- `retry_count`（展示用）

### 入队流程（Notify）
1) 评估规则、匹配渠道。
2) 对每个 channel：
   - 创建一条 `notification_logs`（初始 pending）。
   - 创建一条 `notification_jobs`（初始 queued，`next_run_at=now()`）。
3) 尝试把 job id 推入内存 `workCh`（用于低延迟派发）。
4) 立即返回成功。

### 派发流程（Worker Pool）
- 启动 4 个 worker goroutine 消费 `workCh`。
- 另起 DB 扫描器定时补投递：`status=queued AND next_run_at<=now()`。

每个 job：
1) **Claim**：原子更新 queued -> processing（避免重复处理）。
2) 读取最新 `notification_channels.config` 与模板，执行时渲染（`template/manager.go:93-119`）。
3) 调 provider.Send。
4) 更新 log：sending -> success/failed；写入 sent_at / error_message。
5) 失败时按渠道 retry 策略计算 next_run_at，retry_count++，job 回到 queued 或终态 failed。

## 渠道自定义重试（默认开启）
在 `notification_channels.config` JSON 中新增通用字段：
```json
{
  "retry": {
    "maxAttempts": 5,
    "baseDelay": "5s",
    "maxDelay": "10m",
    "factor": 2,
    "jitter": true
  }
}
```
- Provider 解析时会忽略未知字段（例如 webhook 的 `mapToStruct`：`providers/webhook.go:133-140`），不会破坏现有渠道。
- 未配置则使用默认值（方案 A）。

## 需要调整的 API/前端点
- 统一 status 语义与过滤：建议采用 `0=pending,1=sending,2=success,3=failed`；“全部”用 `status=-1` 或不传。
- header 未读轮询可后续优化：拆分 `unread/count`，弹层展开时拉列表（现状轮询：`header-notification.vue:52-55`）。

## 测试与验收
- 单元测试：
  - rule_engine 并发安全（cache 加锁/改 sync.Map）。
  - Redis nil 情况 frequency evaluator 不 panic。
  - SilenceDuration 生效（触发后应更新内存/或改判定来源）。
- 集成测试（最小）：
  - 入队后立即返回成功；后台最终将 log 变为 success。
  - provider 失败后按渠道策略重试，超过 maxAttempts 终态 failed。

## 后续可选增强（非本次必做）
- 多实例抢占：DB `SKIP LOCKED` 或乐观锁字段。
- 去重/幂等：event_id / dedup_key，降低重复发送概率。
- WebSocket 推送未读通知（docs 已提 TODO）。
