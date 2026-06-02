/**
 * Service_API: WAF release & job management (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/caddy_waf_release.api` / `backend/api/caddy_waf_job.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';
import type { WafKind } from './caddy-source';

// ─── Types ───────────────────────────────────────────────────────────────────

export type WafReleaseStatus = 'downloaded' | 'verified' | 'active' | 'failed' | 'rolled_back';
export type WafJobStatus = 'running' | 'success' | 'failed';

export interface WafReleaseItem {
  id: number;
  sourceId: number;
  kind: WafKind;
  version: string;
  artifactType: string;
  checksum: string;
  sizeBytes: number;
  storagePath: string;
  status: WafReleaseStatus;
  createdAt: string;
  updatedAt: string;
}

export interface WafReleaseListResp {
  list: WafReleaseItem[];
  total: number;
}

export interface WafJobItem {
  id: number;
  sourceId: number;
  releaseId: number;
  action: string;
  triggerMode: string;
  operator: string;
  status: WafJobStatus;
  message: string;
  startedAt: string;
  finishedAt: string;
  createdAt: string;
}

export interface WafJobListResp {
  list: WafJobItem[];
  total: number;
}

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/caddy/waf/release - Fetch WAF release list. */
export function fetchWafReleaseList(params: {
  page: number;
  pageSize: number;
  kind?: WafKind | '';
  status?: WafReleaseStatus | '';
}): Promise<FlatResponse<WafReleaseListResp>> {
  return request<WafReleaseListResp>({ url: '/api/caddy/waf/release', params });
}

/** POST /api/caddy/waf/release/:id/activate - Activate a WAF release. */
export function activateWafRelease(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/release/${id}/activate`, method: 'post' });
}

/** POST /api/caddy/waf/release/rollback - Rollback WAF release. */
export function rollbackWafRelease(data: {
  target?: 'last_good' | 'version';
  version?: string;
}): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/caddy/waf/release/rollback',
    method: 'post',
    data,
  });
}

/** POST /api/caddy/waf/release/clear - Clear WAF releases. */
export function clearWafReleases(data?: { kind?: WafKind | '' }): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/caddy/waf/release/clear',
    method: 'post',
    data,
  });
}

/** GET /api/caddy/waf/job - Fetch WAF job list. */
export function fetchWafJobList(params: {
  page: number;
  pageSize: number;
  status?: WafJobStatus | '';
  action?: string;
}): Promise<FlatResponse<WafJobListResp>> {
  return request<WafJobListResp>({ url: '/api/caddy/waf/job', params });
}

/** POST /api/caddy/waf/job/clear - Clear WAF jobs. */
export function clearWafJobs(): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/caddy/waf/job/clear',
    method: 'post',
  });
}
