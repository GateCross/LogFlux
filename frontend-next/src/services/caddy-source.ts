/**
 * Service_API: WAF source management (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/caddy_waf_source.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

export type WafKind = 'crs' | 'coraza_engine';
export type WafMode = 'remote' | 'manual';
export type WafAuthType = 'none' | 'token' | 'basic';

export interface WafSourceItem {
  id: number;
  name: string;
  kind: WafKind;
  mode: WafMode;
  url: string;
  checksumUrl: string;
  proxyUrl?: string;
  authType: WafAuthType;
  schedule: string;
  enabled: boolean;
  autoCheck: boolean;
  autoDownload: boolean;
  autoActivate: boolean;
  lastRelease: string;
  lastError: string;
  createdAt: string;
  updatedAt: string;
}

export interface WafSourceListResp {
  list: WafSourceItem[];
  total: number;
}

export interface WafSourcePayload {
  name: string;
  kind?: WafKind;
  mode?: WafMode;
  url?: string;
  checksumUrl?: string;
  proxyUrl?: string;
  authType?: WafAuthType;
  authSecret?: string;
  schedule?: string;
  enabled?: boolean;
  autoCheck?: boolean;
  autoDownload?: boolean;
  autoActivate?: boolean;
  meta?: string;
}

export interface WafEngineStatusResp {
  currentVersion?: string;
  latestVersion?: string;
  canUpgrade?: boolean;
  checkedAt?: string;
  source?: string;
  message?: string;
  [key: string]: unknown;
}

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/caddy/waf/source - Fetch WAF source list. */
export function fetchWafSourceList(params: {
  page: number;
  pageSize: number;
  kind?: WafKind | '';
  name?: string;
}): Promise<FlatResponse<WafSourceListResp>> {
  return request<WafSourceListResp>({ url: '/api/caddy/waf/source', params });
}

/** POST /api/caddy/waf/source - Create a new WAF source. */
export function createWafSource(data: WafSourcePayload): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/caddy/waf/source', method: 'post', data });
}

/** PUT /api/caddy/waf/source/:id - Update an existing WAF source. */
export function updateWafSource(id: number, data: Partial<WafSourcePayload>): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/source/${id}`, method: 'put', data });
}

/** DELETE /api/caddy/waf/source/:id - Delete a WAF source. */
export function deleteWafSource(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/source/${id}`, method: 'delete' });
}

/** POST /api/caddy/waf/source/:id/check - Trigger a check for a WAF source. */
export function checkWafSource(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/source/${id}/check`, method: 'post' });
}

/** POST /api/caddy/waf/source/:id/sync - Sync (download + optionally activate) a WAF source. */
export function syncWafSource(id: number, activateNow?: boolean): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/caddy/waf/source/${id}/sync`,
    method: 'post',
    timeout: 240000,
    data: { activateNow },
  });
}

/** POST /api/caddy/waf/upload - Upload a WAF package (FormData). */
export function uploadWafPackage(data: FormData): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/caddy/waf/upload',
    method: 'post',
    data,
  });
}

/** GET /api/caddy/waf/engine/status - Fetch WAF engine status. */
export function fetchWafEngineStatus(): Promise<FlatResponse<WafEngineStatusResp>> {
  return request<WafEngineStatusResp>({ url: '/api/caddy/waf/engine/status' });
}

/** POST /api/caddy/waf/engine/check - Trigger a WAF engine check. */
export function checkWafEngine(): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/caddy/waf/engine/check', method: 'post' });
}
