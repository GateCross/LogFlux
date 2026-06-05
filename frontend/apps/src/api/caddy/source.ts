import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace CaddyWafSourceApi {
  export interface WafSource {
    [key: string]: any;
  }

  export interface WafSourceListResult {
    list: WafSource[];
    total: number;
  }

  export interface WafEngineStatus {
    [key: string]: any;
  }
}

/**
 * 获取 WAF 源列表 — GET /caddy/waf/source
 */
export async function getWafSourceListApi() {
  const resp = await requestClient.get<CaddyWafSourceApi.WafSourceListResult>(
    '/caddy/waf/source',
  );
  return listOf(resp);
}

/**
 * 创建 WAF 源 — POST /caddy/waf/source
 */
export async function createWafSourceApi(data: CaddyWafSourceApi.WafSource) {
  return requestClient.post<void>('/caddy/waf/source', data);
}

/**
 * 更新 WAF 源 — PUT /caddy/waf/source/:id
 */
export async function updateWafSourceApi(
  id: number,
  data: CaddyWafSourceApi.WafSource,
) {
  return requestClient.put<void>(`/caddy/waf/source/${id}`, data);
}

/**
 * 删除 WAF 源 — DELETE /caddy/waf/source/:id
 */
export async function deleteWafSourceApi(id: number) {
  return requestClient.delete<void>(`/caddy/waf/source/${id}`);
}

/**
 * 检查 WAF 源更新 — POST /caddy/waf/source/:id/check
 */
export async function checkWafSourceApi(id: number) {
  return requestClient.post<void>(`/caddy/waf/source/${id}/check`);
}

/**
 * 从 WAF 源同步/下载（超时 240 秒） — POST /caddy/waf/source/:id/sync
 */
export async function syncWafSourceApi(id: number) {
  return requestClient.post<void>(
    `/caddy/waf/source/${id}/sync`,
    undefined,
    { timeout: 240_000 },
  );
}

/**
 * 上传 WAF 包（FormData） — POST /caddy/waf/upload
 */
export async function uploadWafPackageApi(formData: FormData) {
  return requestClient.post<void>('/caddy/waf/upload', formData);
}

/**
 * 获取 WAF 引擎状态 — GET /caddy/waf/engine/status
 */
export async function getWafEngineStatusApi() {
  return requestClient.get<CaddyWafSourceApi.WafEngineStatus>(
    '/caddy/waf/engine/status',
  );
}

/**
 * 检查 WAF 引擎更新 — POST /caddy/waf/engine/check
 */
export async function checkWafEngineUpdateApi() {
  return requestClient.post<void>('/caddy/waf/engine/check');
}
