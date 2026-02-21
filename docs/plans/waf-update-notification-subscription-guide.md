# LogFlux WAF 更新事件订阅建议

> 适用版本：2026-02-21 之后  
> 关联文档：`docs/plans/waf-update-operations-guide.md`

## 1. 建议启用的事件

- `security.waf_source_check_failed`
- `security.waf_source_sync_failed`
- `security.waf_release_activate_failed`
- `security.waf_release_rollback_success`
- `security.waf_release_rollback_failed`
- `security.waf_source_sync_success`（建议仅值班群）

## 2. 推荐分级

- `P1`（立即告警）：`*_failed`
- `P2`（关注）：`waf_release_rollback_success`
- `P3`（信息）：`waf_source_sync_success`

## 3. 推荐订阅策略

- 渠道 A（值班群）：订阅全部 `security.waf_*`
- 渠道 B（管理群）：仅订阅 `security.waf_*_failed`
- 渠道 C（审计归档）：订阅全部 `security.waf_*`，并保留 90 天

## 4. 规则示例

- 规则 1：`security.waf_release_activate_failed` 触发即告警，无静默
- 规则 2：10 分钟内 `security.waf_source_sync_failed` >= 3 次触发升级告警
- 规则 3：`security.waf_release_rollback_success` 触发后自动创建复盘工单

## 5. 上线检查项

- 是否存在至少 1 条 `security.waf_*_failed` 的订阅规则
- 是否能在通知 payload 中看到 `sourceId/releaseId/jobId/triggerMode`
- 是否有值班人员覆盖非工作时段
