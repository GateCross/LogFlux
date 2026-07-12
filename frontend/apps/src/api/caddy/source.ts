import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace CaddyWafSourceApi {
  /** WAF 源列表项 — 对齐 backend WafSourceItem */
  export interface WafSource {
    id: number;
    name: string;
    kind: string;
    mode: string;
    url: string;
    checksumUrl?: string;
    proxyUrl?: string;
    authType?: string;
    schedule?: string;
    enabled?: boolean;
    autoCheck?: boolean;
    autoDownload?: boolean;
    autoActivate?: boolean;
    lastRelease?: string;
    lastError?: string;
    createdAt?: string;
    updatedAt?: string;
    /** 页面历史字段：部分 UI 仍用 type 展示 kind */
    type?: string;
    status?: string;
  }

  /** 创建/更新载荷（页面表单字段） */
  export interface WafSourceWrite {
    name: string;
    url?: string;
    kind?: string;
    mode?: string;
    type?: string;
    enabled?: boolean;
  }

  export interface WafSourceListResult {
    list: WafSource[];
    total: number;
  }

  /** WAF 引擎状态 — 对齐 backend WafEngineStatusResp */
  export interface WafEngineStatus {
    serverId: number;
    currentVersion?: string;
    latestVersion?: string;
    canUpgrade: boolean;
    checkedAt?: string;
    source?: string;
    message?: string;
  }
}

/**
 * 获取 WAF 源列表 — GET /caddy/waf/source
 */
export async function getWafSourceListApi(config?: RequestClientConfig) {
  const resp = await requestClient.get<CaddyWafSourceApi.WafSourceListResult>(
    '/caddy/waf/source',
    config,
  );
  return listOf(resp);
}

/**
 * 创建 WAF 源 — POST /caddy/waf/source
 */
export async function createWafSourceApi(
  data: CaddyWafSourceApi.WafSourceWrite,
) {
  return requestClient.post<void>('/caddy/waf/source', {
    name: data.name,
    url: data.url,
    kind: data.kind ?? data.type ?? 'crs',
    mode: data.mode ?? 'remote',
    enabled: data.enabled,
  });
}

/**
 * 更新 WAF 源 — PUT /caddy/waf/source/:id
 */
export async function updateWafSourceApi(
  id: number,
  data: CaddyWafSourceApi.WafSourceWrite,
) {
  return requestClient.put<void>(`/caddy/waf/source/${id}`, {
    name: data.name,
    url: data.url,
    kind: data.kind ?? data.type,
    mode: data.mode,
    enabled: data.enabled,
  });
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
export async function getWafEngineStatusApi(config?: RequestClientConfig) {
  return requestClient.get<CaddyWafSourceApi.WafEngineStatus>(
    '/caddy/waf/engine/status',
    config,
  );
}

/**
 * 检查 WAF 引擎更新 — POST /caddy/waf/engine/check
 */
export async function checkWafEngineUpdateApi() {
  return requestClient.post<void>('/caddy/waf/engine/check');
}
