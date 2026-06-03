import { requestClient } from '#/api/request';

export namespace CaddyWafReleaseApi {
  export interface WafRelease {
    [key: string]: any;
  }

  export interface WafJob {
    [key: string]: any;
  }
}

// ─── WAF Releases ───

/**
 * 获取 WAF 发布列表 — GET /caddy/waf/release
 */
export async function getWafReleaseListApi() {
  return requestClient.get<CaddyWafReleaseApi.WafRelease[]>(
    '/caddy/waf/release',
  );
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
  return requestClient.get<CaddyWafReleaseApi.WafJob[]>('/caddy/waf/job');
}

/**
 * 清除任务历史 — POST /caddy/waf/job/clear
 */
export async function clearWafJobHistoryApi() {
  return requestClient.post<void>('/caddy/waf/job/clear');
}
