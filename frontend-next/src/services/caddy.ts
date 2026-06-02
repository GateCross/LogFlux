/**
 * Service_API: Caddy server/config management (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/caddy.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

export interface CaddyServer {
  id: number;
  name: string;
  url: string;
  type: string;
  token?: string;
}

export interface CaddyServerListResp {
  list: CaddyServer[];
}

export interface CaddyConfigResp {
  config: string;
  modules?: string;
}

export interface CaddyConfigPreviewResp {
  valid: boolean;
  config: string;
  errors: string[];
  actions: string[];
}

export interface CaddyConfigHistoryItem {
  id: number;
  serverId: number;
  action: string;
  hash: string;
  createdAt: string;
}

export interface CaddyConfigHistoryListResp {
  list: CaddyConfigHistoryItem[];
  total: number;
}

export interface CaddyConfigHistoryDetailResp {
  id: number;
  serverId: number;
  action: string;
  hash: string;
  config: string;
  createdAt: string;
}

export interface CaddyLogItem {
  id: number;
  logTime: string;
  country: string;
  province?: string;
  city: string;
  location?: string;
  host: string;
  method: string;
  uri: string;
  status: number;
  size: number;
  remoteIp: string;
  clientIp: string;
  userAgent: string;
  duration: number;
  bodyBytes: number;
  rawLog: string;
}

export interface CaddyLogListResp {
  list: CaddyLogItem[];
  total: number;
}

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/caddy/server - Fetch all Caddy servers. */
export function fetchCaddyServers(): Promise<FlatResponse<CaddyServerListResp>> {
  return request<CaddyServerListResp>({ url: '/api/caddy/server' });
}

/** POST /api/caddy/server - Add a new Caddy server. */
export function addCaddyServer(data: Omit<CaddyServer, 'id'>): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/caddy/server', method: 'post', data });
}

/** PUT /api/caddy/server/:id - Update an existing Caddy server. */
export function updateCaddyServer(data: CaddyServer): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/server/${data.id}`, method: 'put', data });
}

/** DELETE /api/caddy/server/:id - Delete a Caddy server. */
export function deleteCaddyServer(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/server/${id}`, method: 'delete' });
}

/** GET /api/caddy/server/:serverId/config - Fetch Caddy config for a server. */
export function fetchCaddyConfig(serverId: number): Promise<FlatResponse<CaddyConfigResp>> {
  return request<CaddyConfigResp>({ url: `/api/caddy/server/${serverId}/config` });
}

/** POST /api/caddy/server/:serverId/config (mode: raw) - Update config in raw mode. */
export function updateCaddyConfigRaw(serverId: number, config: string): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/caddy/server/${serverId}/config`,
    method: 'post',
    data: { mode: 'raw', config },
  });
}

/** POST /api/caddy/server/:serverId/config (mode: quick) - Update config in structured/quick mode. */
export function updateCaddyConfigStructured(
  serverId: number,
  config: string,
  modules: string,
): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/caddy/server/${serverId}/config`,
    method: 'post',
    data: { mode: 'quick', config, modules },
  });
}

/** POST /api/caddy/server/:serverId/config/preview - Preview config changes. */
export function previewCaddyConfig(
  serverId: number,
  data: { mode?: 'quick' | 'raw'; config?: string; modules?: string },
): Promise<FlatResponse<CaddyConfigPreviewResp>> {
  return request<CaddyConfigPreviewResp>({
    url: `/api/caddy/server/${serverId}/config/preview`,
    method: 'post',
    data,
  });
}

/** GET /api/caddy/logs - Fetch Caddy access logs with pagination/filters. */
export function fetchCaddyLogs(params: {
  page: number;
  pageSize: number;
  keyword?: string;
  host?: string;
  status?: number;
  startTime?: string;
  endTime?: string;
  sortBy?: string;
  order?: string;
}): Promise<FlatResponse<CaddyLogListResp>> {
  return request<CaddyLogListResp>({ url: '/api/caddy/logs', params });
}

/** GET /api/caddy/server/:serverId/config/history - Fetch config change history. */
export function fetchCaddyConfigHistory(
  serverId: number,
  params: { page: number; pageSize: number },
): Promise<FlatResponse<CaddyConfigHistoryListResp>> {
  return request<CaddyConfigHistoryListResp>({
    url: `/api/caddy/server/${serverId}/config/history`,
    params,
  });
}

/** GET /api/caddy/server/:serverId/config/history/:historyId - Fetch config history detail. */
export function fetchCaddyConfigHistoryDetail(
  serverId: number,
  historyId: number,
): Promise<FlatResponse<CaddyConfigHistoryDetailResp>> {
  return request<CaddyConfigHistoryDetailResp>({
    url: `/api/caddy/server/${serverId}/config/history/${historyId}`,
  });
}

/** POST /api/caddy/server/:serverId/config/rollback - Rollback config to a history snapshot. */
export function rollbackCaddyConfig(
  serverId: number,
  historyId: number,
): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/caddy/server/${serverId}/config/rollback`,
    method: 'post',
    data: { historyId },
  });
}
