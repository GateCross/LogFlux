import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace CaddyServerApi {
  /**
   * Caddy 服务器列表项 — 对齐 backend CaddyServerItem
   * 写请求字段见 CaddyServerCreate / CaddyServerUpdate，勿回写索引签名
   */
  export interface CaddyServer {
    id: number;
    name: string;
    url: string;
    type: string;
    createdAt: string;
  }

  /** 新增服务器 — 对齐 backend CaddyServerReq */
  export interface CaddyServerCreate {
    name: string;
    url: string;
    token?: string;
    /** 默认 local；local | remote */
    type?: string;
    username?: string;
    password?: string;
  }

  /** 更新服务器 — 对齐 backend UpdateCaddyServerReq（path id 由 API 函数单独传） */
  export interface CaddyServerUpdate {
    name?: string;
    url?: string;
    token?: string;
    type?: string;
    username?: string;
    password?: string;
  }

  export interface CaddyServerListResult {
    list: CaddyServer[];
  }

  /**
   * 配置读取响应 — 对齐 backend CaddyConfigResp
   * 注意：预览/保存请求体请用 CaddyConfigWrite，勿与读响应混用
   */
  export interface CaddyConfig {
    config: string;
    modules?: string;
  }

  /**
   * 配置预览/保存请求体 — 对齐 backend CaddyConfigUpdateReq / CaddyConfigPreviewReq
   * （path serverId 由 API 函数单独传）
   */
  export interface CaddyConfigWrite {
    /** quick | raw；为空时后端按 modules 自动判断 / 兼容旧版 */
    mode?: string;
    /** 原始 Caddyfile 或前端兜底生成结果 */
    config?: string;
    /** 结构化配置(JSON) */
    modules?: string;
  }

  /** 配置预览响应 — 对齐 backend CaddyConfigPreviewResp */
  export interface ConfigPreview {
    valid: boolean;
    config: string;
    errors: string[];
    actions: string[];
  }

  /** 配置历史列表项 — 对齐 backend CaddyConfigHistoryItem（无 config/modules 正文） */
  export interface ConfigHistoryItem {
    id: number;
    serverId: number;
    action: string;
    hash: string;
    createdAt: string;
  }

  /**
   * 配置历史详情 — 对齐 backend CaddyConfigHistoryDetailResp
   * 与 ConfigHistoryItem 分离，避免列表项被当成含全文
   */
  export interface ConfigHistoryDetail extends ConfigHistoryItem {
    config: string;
    modules?: string;
  }

  export interface ConfigHistoryListResult {
    list: ConfigHistoryItem[];
    total: number;
  }

  export interface RollbackParams {
    historyId: number;
  }

  /**
   * Caddy 访问日志项 — 对齐 backend CaddyLogItem。
   * 保持原始日志为字符串，页面按需解析展示，不以开放字典承接未知字段。
   */
  export interface CaddyLogItem {
    id: number;
    logTime: string;
    country: string;
    province: string;
    city: string;
    location: string;
    host: string;
    method: string;
    uri: string;
    status: number;
    size: number;
    remoteIp: string;
    clientIp: string;
    userAgent: string;
    rawLog: string;
  }

  /** CaddyLogReq 的受限查询参数；未传字段由后端默认值处理。 */
  export interface CaddyLogQuery {
    endTime?: string;
    host?: string;
    keyword?: string;
    order?: 'asc' | 'desc';
    page?: number;
    pageSize?: number;
    sortBy?: 'logTime';
    startTime?: string;
    /** -1 表示不过滤状态码。 */
    status?: number;
  }

  export interface CaddyLogPageResult {
    list: CaddyLogItem[];
    total: number;
  }

  /** 节点探测结果 — GET /caddy/server/status；对齐 CaddyServerStatusItem */
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
export async function getCaddyServerListApi(config?: RequestClientConfig) {
  const resp = await requestClient.get<CaddyServerApi.CaddyServerListResult>(
    '/caddy/server',
    config,
  );
  return listOf(resp);
}

/**
 * 探测 Caddy 节点在线状态（只读，不触发 /load）— GET /caddy/server/status
 */
export async function getCaddyServerStatusApi(config?: RequestClientConfig) {
  const resp = await requestClient.get<CaddyServerApi.CaddyServerStatusListResult>(
    '/caddy/server/status',
    config,
  );
  return listOf(resp);
}

/**
 * 添加 Caddy 服务器 — POST /caddy/server
 */
export async function addCaddyServerApi(data: CaddyServerApi.CaddyServerCreate) {
  return requestClient.post<void>('/caddy/server', data);
}

/**
 * 更新 Caddy 服务器 — PUT /caddy/server/:id
 */
export async function updateCaddyServerApi(
  id: number,
  data: CaddyServerApi.CaddyServerUpdate,
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
export async function getCaddyConfigApi(
  serverId: number,
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddyServerApi.CaddyConfig>(
    `/caddy/server/${serverId}/config`,
    config,
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
    { config, mode: modules ? 'quick' : 'raw', modules } satisfies CaddyServerApi.CaddyConfigWrite,
  );
}

/**
 * 预览 Caddy 配置变更 — POST /caddy/server/:serverId/config/preview
 */
export async function previewCaddyConfigApi(
  serverId: number,
  data: CaddyServerApi.CaddyConfigWrite,
) {
  return requestClient.post<CaddyServerApi.ConfigPreview>(
    `/caddy/server/${serverId}/config/preview`,
    data,
  );
}

/**
 * 获取配置历史列表 — GET /caddy/server/:serverId/config/history
 */
export async function getCaddyConfigHistoryListApi(
  serverId: number,
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddyServerApi.ConfigHistoryListResult>(
    `/caddy/server/${serverId}/config/history`,
    config,
  );
}

/**
 * 获取配置历史详情 — GET /caddy/server/:serverId/config/history/:historyId
 */
export async function getCaddyConfigHistoryDetailApi(
  serverId: number,
  historyId: number,
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddyServerApi.ConfigHistoryDetail>(
    `/caddy/server/${serverId}/config/history/${historyId}`,
    config,
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
export async function getCaddyLogsApi(
  params?: CaddyServerApi.CaddyLogQuery,
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddyServerApi.CaddyLogPageResult>('/caddy/logs', {
    params,
    ...config,
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
export async function getSiteMetricsApi(
  params: SiteMetricsParams,
  config?: RequestClientConfig,
) {
  const resp = await requestClient.post<SiteMetricsListResult>(
    '/caddy/logs/site-metrics',
    {
      hosts: params.hosts,
      windowMinutes: params.windowMinutes,
    },
    config,
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
