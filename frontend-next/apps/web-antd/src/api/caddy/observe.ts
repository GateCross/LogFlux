import { requestClient } from '#/api/request';

export namespace CaddyWafObserveApi {
  export interface WafPolicyStats {
    [key: string]: any;
  }

  export interface FalsePositiveFeedback {
    [key: string]: any;
  }

  export interface FeedbackStatusUpdate {
    status: string;
  }

  export interface FeedbackBatchStatusUpdate {
    ids: number[];
    status: string;
  }
}

/**
 * 获取 WAF 策略命中/拦截统计 — GET /caddy/waf/policy/stats
 */
export async function getWafPolicyStatsApi(params?: Record<string, any>) {
  return requestClient.get<CaddyWafObserveApi.WafPolicyStats>(
    '/caddy/waf/policy/stats',
    { params },
  );
}

/**
 * 获取误报反馈列表 — GET /caddy/waf/policy/false-positive-feedback
 */
export async function getFalsePositiveFeedbackListApi(
  params?: Record<string, any>,
) {
  return requestClient.get<CaddyWafObserveApi.FalsePositiveFeedback[]>(
    '/caddy/waf/policy/false-positive-feedback',
    { params },
  );
}

/**
 * 提交误报反馈 — POST /caddy/waf/policy/false-positive-feedback
 */
export async function submitFalsePositiveFeedbackApi(
  data: CaddyWafObserveApi.FalsePositiveFeedback,
) {
  return requestClient.post<void>(
    '/caddy/waf/policy/false-positive-feedback',
    data,
  );
}

/**
 * 更新误报反馈状态 — PUT /caddy/waf/policy/false-positive-feedback/:id/status
 */
export async function updateFeedbackStatusApi(
  id: number,
  data: CaddyWafObserveApi.FeedbackStatusUpdate,
) {
  return requestClient.put<void>(
    `/caddy/waf/policy/false-positive-feedback/${id}/status`,
    data,
  );
}

/**
 * 批量更新误报反馈状态 — PUT /caddy/waf/policy/false-positive-feedback/batch-status
 */
export async function batchUpdateFeedbackStatusApi(
  data: CaddyWafObserveApi.FeedbackBatchStatusUpdate,
) {
  return requestClient.put<void>(
    '/caddy/waf/policy/false-positive-feedback/batch-status',
    data,
  );
}
