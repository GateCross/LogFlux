import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace CaddyServerApi {
  export interface CaddyServer {
    [key: string]: any;
  }

  export interface CaddyServerListResult {
    list: CaddyServer[];
  }

  export interface CaddyConfig {
    [key: string]: any;
  }

  export interface ConfigPreview {
    [key: string]: any;
  }

  export interface ConfigHistoryItem {
    [key: string]: any;
  }

  export interface ConfigHistoryListResult {
    list: ConfigHistoryItem[];
    total: number;
  }

  export interface RollbackParams {
    historyId: number;
  }

  export interface CaddyLogItem {
    [key: string]: any;
  }

  export interface CaddyLogPageResult {
    list: CaddyLogItem[];
    total: number;
  }

  /** 节点探测结果 — GET /caddy/server/status */
  export interface CaddyServerStatusItem {
    serverId: number;
    name: string;
    reachable: boolean;
    latencyMs: number;
    probedAt: string;
    errorMessage?: string;
  }

  export interface CaddyServerStatusListResult {
    list: CaddyServerStatusItem[];
  }
}

/**
 * 获取 Caddy 服务器列表 — GET /caddy/server
 */
export async function getCaddyServerListApi() {
  const resp = await requestClient.get<CaddyServerApi.CaddyServerListResult>(
    '/caddy/server',
  );
  return listOf(resp);
}

/**
 * 探测 Caddy 节点在线状态（只读，不触发 /load）— GET /caddy/server/status
 */
export async function getCaddyServerStatusApi() {
  const resp = await requestClient.get<CaddyServerApi.CaddyServerStatusListResult>(
    '/caddy/server/status',
  );
  return listOf(resp);
}

/**
 * 添加 Caddy 服务器 — POST /caddy/server
 */
export async function addCaddyServerApi(data: CaddyServerApi.CaddyServer) {
  return requestClient.post<void>('/caddy/server', data);
}

/**
 * 更新 Caddy 服务器 — PUT /caddy/server/:id
 */
export async function updateCaddyServerApi(
  id: number,
  data: CaddyServerApi.CaddyServer,
) {
  return requestClient.put<void>(`/caddy/server/${id}`, data);
}

/**
 * 删除 Caddy 服务器 — DELETE /caddy/server/:id
 */
export async function deleteCaddyServerApi(id: number) {
  return requestClient.delete<void>(`/caddy/server/${id}`);
}

/**
 * 获取 Caddy 配置 — GET /caddy/server/:serverId/config
 */
export async function getCaddyConfigApi(serverId: number) {
  return requestClient.get<CaddyServerApi.CaddyConfig>(
    `/caddy/server/${serverId}/config`,
  );
}

/**
 * 推送 Caddy 原始配置 — POST /caddy/server/:serverId/config
 */
export async function pushCaddyConfigApi(
  serverId: number,
  config: string,
  modules?: string,
) {
  return requestClient.post<void>(
    `/caddy/server/${serverId}/config`,
    { config, mode: modules ? 'quick' : 'raw', modules },
  );
}

/**
 * 预览 Caddy 配置变更 — POST /caddy/server/:serverId/config/preview
 */
export async function previewCaddyConfigApi(
  serverId: number,
  data: CaddyServerApi.CaddyConfig,
) {
  return requestClient.post<CaddyServerApi.ConfigPreview>(
    `/caddy/server/${serverId}/config/preview`,
    data,
  );
}

/**
 * 获取配置历史列表 — GET /caddy/server/:serverId/config/history
 */
export async function getCaddyConfigHistoryListApi(serverId: number) {
  return requestClient.get<CaddyServerApi.ConfigHistoryListResult>(
    `/caddy/server/${serverId}/config/history`,
  );
}

/**
 * 获取配置历史详情 — GET /caddy/server/:serverId/config/history/:historyId
 */
export async function getCaddyConfigHistoryDetailApi(
  serverId: number,
  historyId: number,
) {
  return requestClient.get<CaddyServerApi.ConfigHistoryItem>(
    `/caddy/server/${serverId}/config/history/${historyId}`,
  );
}

/**
 * 回滚 Caddy 配置 — POST /caddy/server/:serverId/config/rollback
 */
export async function rollbackCaddyConfigApi(
  serverId: number,
  data: CaddyServerApi.RollbackParams,
) {
  return requestClient.post<void>(
    `/caddy/server/${serverId}/config/rollback`,
    data,
  );
}

/**
 * 查询 Caddy 访问日志（分页） — GET /caddy/logs
 */
export async function getCaddyLogsApi(params?: Record<string, any>) {
  return requestClient.get<CaddyServerApi.CaddyLogPageResult>('/caddy/logs', {
    params,
  });
}

/** 站点近窗 4xx/5xx 指标项 — POST /caddy/logs/site-metrics */
export interface SiteMetricsItem {
  host: string;
  count4xx: number;
  count5xx: number;
}

export interface SiteMetricsListResult {
  list: SiteMetricsItem[];
}

export interface SiteMetricsParams {
  /** 主机列表，单次最多 50 */
  hosts: string[];
  /** 时间窗分钟数，默认 15 */
  windowMinutes?: number;
}

/**
 * 批量查询站点近窗 4xx/5xx 指标 — POST /caddy/logs/site-metrics
 * 只读聚合，不触发 /load；失败时由调用方降级展示。
 */
export async function getSiteMetricsApi(params: SiteMetricsParams) {
  const resp = await requestClient.post<SiteMetricsListResult>(
    '/caddy/logs/site-metrics',
    {
      hosts: params.hosts,
      windowMinutes: params.windowMinutes,
    },
  );
  return listOf(resp);
}

/**
 * 清空 Caddy 访问日志 — POST /caddy/logs/clear
 */
export async function clearCaddyLogsApi() {
  return requestClient.post<void>('/caddy/logs/clear');
}

/** Docker 发现候选项 — GET /caddy/discovery/docker（会话候选，不落库、不 /load） */
export interface DockerDiscoveryCandidate {
  candidateId: string;
  containerId: string;
  containerName: string;
  status: string;
  name: string;
  domains: string[];
  upstream: string;
  lbPolicy?: string;
  tlsMode: string;
  healthPath?: string;
  healthInterval?: string;
  healthTimeout?: string;
  reason?: string;
  valid: boolean;
}

export interface DockerDiscoveryResult {
  list: DockerDiscoveryCandidate[];
  scannedAt: string;
  message?: string;
}

/**
 * 扫描 Docker 容器 labels，返回会话候选草稿数据。
 * 只读：不调用 /load，不写 discovery DB。
 */
export async function discoverDockerServicesApi() {
  return requestClient.get<DockerDiscoveryResult>('/caddy/discovery/docker');
}
