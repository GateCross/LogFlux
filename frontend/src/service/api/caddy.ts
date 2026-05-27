import { request } from '../request';

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
  method: string;
  uri: string;
  host: string;
  status: number;
  remoteIp: string;
  userAgent: string;
  duration: number;
  bodyBytes: number;
}

export interface CaddyLogListResp {
  list: CaddyLogItem[];
  total: number;
}

export function fetchCaddyServers() {
  return request<CaddyServerListResp>({ url: '/api/caddy/server' });
}

export function addCaddyServer(data: Omit<CaddyServer, 'id'>) {
  return request<void>({ url: '/api/caddy/server', method: 'post', data });
}

export function updateCaddyServer(data: CaddyServer) {
  return request<void>({ url: `/api/caddy/server/${data.id}`, method: 'put', data });
}

export function deleteCaddyServer(id: number) {
  return request<void>({ url: `/api/caddy/server/${id}`, method: 'delete' });
}

export function fetchCaddyConfig(serverId: number) {
  return request<CaddyConfigResp>({ url: `/api/caddy/server/${serverId}/config` });
}

export function updateCaddyConfigRaw(serverId: number, config: string) {
  return request<void>({
    url: `/api/caddy/server/${serverId}/config`,
    method: 'post',
    data: { mode: 'raw', config }
  });
}

export function updateCaddyConfigStructured(serverId: number, config: string, modules: string) {
  return request<void>({
    url: `/api/caddy/server/${serverId}/config`,
    method: 'post',
    data: { mode: 'quick', config, modules }
  });
}

export function previewCaddyConfig(
  serverId: number,
  data: { mode?: 'quick' | 'raw'; config?: string; modules?: string }
) {
  return request<CaddyConfigPreviewResp>({
    url: `/api/caddy/server/${serverId}/config/preview`,
    method: 'post',
    data
  });
}

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
}) {
  return request<CaddyLogListResp>({ url: '/api/caddy/logs', params });
}

export function fetchCaddyConfigHistory(serverId: number, params: { page: number; pageSize: number }) {
  return request<CaddyConfigHistoryListResp>({ url: `/api/caddy/server/${serverId}/config/history`, params });
}

export function fetchCaddyConfigHistoryDetail(serverId: number, historyId: number) {
  return request<CaddyConfigHistoryDetailResp>({ url: `/api/caddy/server/${serverId}/config/history/${historyId}` });
}

export function rollbackCaddyConfig(serverId: number, historyId: number) {
  return request<void>({
    url: `/api/caddy/server/${serverId}/config/rollback`,
    method: 'post',
    data: { historyId }
  });
}

export * from './caddy-source';
export * from './caddy-policy';
export * from './caddy-observe';
export * from './caddy-release-job';
export * from './caddy-simple-waf';
