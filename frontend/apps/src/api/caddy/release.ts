import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace CaddyWafReleaseApi {
  export interface WafRelease {
    [key: string]: any;
  }

  export interface WafReleaseListResult {
    list: WafRelease[];
    total: number;
  }

  export interface WafJob {
    [key: string]: any;
  }

  export interface WafJobListResult {
    list: WafJob[];
    total: number;
  }
}

// ─── WAF Releases ───

/**
 * 获取 WAF 发布列表 — GET /caddy/waf/release
 */
export async function getWafReleaseListApi() {
  const resp = await requestClient.get<CaddyWafReleaseApi.WafReleaseListResult>(
    '/caddy/waf/release',
  );
  return listOf(resp);
}

/**
 * 激活 WAF 发布 — POST /caddy/waf/release/:id/activate
 */
export async function activateWafReleaseApi(id: number) {
  return requestClient.post<void>(`/caddy/waf/release/${id}/activate`);
}

/**
 * 回滚 WAF 发布 — POST /caddy/waf/release/rollback
 */
export async function rollbackWafReleaseApi() {
  return requestClient.post<void>('/caddy/waf/release/rollback');
}

/**
 * 清除发布历史 — POST /caddy/waf/release/clear
 */
export async function clearWafReleaseHistoryApi() {
  return requestClient.post<void>('/caddy/waf/release/clear');
}

// ─── WAF Jobs ───

/**
 * 获取 WAF 任务列表 — GET /caddy/waf/job
 */
export async function getWafJobListApi() {
  const resp = await requestClient.get<CaddyWafReleaseApi.WafJobListResult>(
    '/caddy/waf/job',
  );
  return listOf(resp);
}

/**
 * 清除任务历史 — POST /caddy/waf/job/clear
 */
export async function clearWafJobHistoryApi() {
  return requestClient.post<void>('/caddy/waf/job/clear');
}
