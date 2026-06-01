/**
 * Security 模块共享类型（task 11.1 / design.md「Data Models」）。
 *
 * 集中存放 WAF 相关枚举与数据模型，供 `src/services/security/*` 的纯逻辑模块
 * （mappers / policy-utils / validators 等）共享引用，避免重复定义。
 *
 * 类型来源（旧 Vue 版）：
 *  - `src/service/api/caddy-policy.ts`：策略引擎模式 / CRS 模板 / 作用域 / 版本状态
 *  - `src/service/api/caddy-release-job.ts`：发布状态 / 任务状态
 *  - `src/service/api/caddy-observe.ts`：误报反馈项
 *
 * 命名与 design.md Data Models 保持一致，便于其他 security 纯逻辑模块（如 task 11.2
 * 的 validators）按相同名称导入复用。各联合成员与旧版逐一对齐以保证移植后行为等价。
 */

/** WAF 策略引擎模式（对应旧 `caddy-policy.ts`）。 */
export type WafPolicyEngineMode = 'on' | 'off' | 'detectiononly';

/** WAF 策略 CRS 模板（对应旧 `caddy-policy.ts`）。 */
export type WafPolicyCrsTemplate = 'low_fp' | 'balanced' | 'high_blocking' | 'custom';

/** WAF 策略版本状态（对应旧 `caddy-policy.ts`）。 */
export type WafPolicyRevisionStatus = 'draft' | 'published' | 'rolled_back';

/** WAF 策略作用域类型（对应旧 `caddy-policy.ts`）。 */
export type WafPolicyScopeType = 'global' | 'site' | 'route';

/** WAF 排除项移除类型（对应旧 `caddy-policy.ts`）。 */
export type WafPolicyRemoveType = 'id' | 'tag';

/** WAF 发布状态（对应旧 `caddy-release-job.ts`）。 */
export type WafReleaseStatus = 'downloaded' | 'verified' | 'active' | 'failed' | 'rolled_back';

/** WAF 任务状态（对应旧 `caddy-release-job.ts`）。 */
export type WafJobStatus = 'running' | 'success' | 'failed';

/**
 * WAF 策略误报反馈项（对应旧 `caddy-observe.ts` 的 `WafPolicyFalsePositiveFeedbackItem`）。
 *
 * security-mappers 的 SLA 映射函数依赖其 `feedbackStatus` 与 `isOverdue` 字段。
 */
export interface WafPolicyFalsePositiveFeedbackItem {
  id: number;
  policyId: number;
  policyName: string;
  host: string;
  path: string;
  method: string;
  status: number;
  feedbackStatus: 'pending' | 'confirmed' | 'resolved';
  assignee: string;
  dueAt: string;
  isOverdue: boolean;
  sampleUri: string;
  reason: string;
  suggestion: string;
  operator: string;
  processNote: string;
  processedBy: string;
  processedAt: string;
  createdAt: string;
}
