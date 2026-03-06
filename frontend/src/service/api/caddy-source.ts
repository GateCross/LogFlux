import { request } from '../request';

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

export function fetchWafSourceList(params: { page: number; pageSize: number; kind?: WafKind | ''; name?: string }) {
  return request<WafSourceListResp>({ url: '/api/caddy/waf/source', params });
}

export function createWafSource(data: WafSourcePayload) {
  return request<any>({ url: '/api/caddy/waf/source', method: 'post', data });
}

export function updateWafSource(id: number, data: Partial<WafSourcePayload>) {
  return request<any>({ url: `/api/caddy/waf/source/${id}`, method: 'put', data });
}

export function deleteWafSource(id: number) {
  return request<any>({ url: `/api/caddy/waf/source/${id}`, method: 'delete' });
}

export function checkWafSource(id: number) {
  return request<any>({ url: `/api/caddy/waf/source/${id}/check`, method: 'post' });
}

export function syncWafSource(id: number, activateNow?: boolean) {
  return request<any>({
    url: `/api/caddy/waf/source/${id}/sync`,
    method: 'post',
    timeout: 240000,
    data: { activateNow }
  });
}

export function uploadWafPackage(data: FormData) {
  return request<any>({
    url: '/api/caddy/waf/upload',
    method: 'post',
    data
  });
}

export function fetchWafEngineStatus() {
  return request<WafEngineStatusResp>({ url: '/api/caddy/waf/engine/status' });
}

export function checkWafEngine() {
  return request<any>({ url: '/api/caddy/waf/engine/check', method: 'post' });
}
