import { request } from '../request';
import type { WafKind } from './caddy-source';

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

export function fetchWafReleaseList(params: {
  page: number;
  pageSize: number;
  kind?: WafKind | '';
  status?: WafReleaseStatus | '';
}) {
  return request<WafReleaseListResp>({ url: '/api/caddy/waf/release', params });
}

export function activateWafRelease(id: number) {
  return request<any>({ url: `/api/caddy/waf/release/${id}/activate`, method: 'post' });
}

export function rollbackWafRelease(data: { target?: 'last_good' | 'version'; version?: string }) {
  return request<any>({
    url: '/api/caddy/waf/release/rollback',
    method: 'post',
    data
  });
}

export function clearWafReleases(data?: { kind?: WafKind | '' }) {
  return request<any>({
    url: '/api/caddy/waf/release/clear',
    method: 'post',
    data
  });
}

export function fetchWafJobList(params: { page: number; pageSize: number; status?: WafJobStatus | ''; action?: string }) {
  return request<WafJobListResp>({ url: '/api/caddy/waf/job', params });
}

export function clearWafJobs() {
  return request<any>({
    url: '/api/caddy/waf/job/clear',
    method: 'post'
  });
}
