import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace CaddyServerApi {
  /**
   * 配置读取响应 — 对齐 backend CaddyConfigResp
   * 注意：预览/保存请求体请用 CaddyConfigWrite，勿与读响应混用
   */
  export interface CaddyConfig {
    config: string;
    modules?: string;
    /** 当前库内配置哈希；写请求回传做乐观并发 */
    hash?: string;
  }

  /**
   * 配置预览/保存请求体 — 对齐 backend CaddyConfigUpdateReq / CaddyConfigPreviewReq
   * 单实例：没有 serverId 维度，目标实例由后端自行解析
   */
  export interface CaddyConfigWrite {
    /** quick | raw；为空时后端按 modules 自动判断 / 兼容旧版 */
    mode?: string;
    /** 原始 Caddyfile 或前端兜底生成结果 */
    config?: string;
    /** 结构化配置(JSON) */
    modules?: string;
    /** 编辑起点的配置哈希；与库内不一致时后端拒绝写入 */
    baseHash?: string;
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
    /** 回滚起点配置哈希；与库内不一致时后端拒绝回滚 */
    baseHash?: string;
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
    /** 日志级别；错误日志视图用于区分严重程度 */
    level?: string;
    /** logger 命名空间；错误日志视图用于定位来源模块 */
    logger?: string;
    rawLog: string;
    /** WAF 审计元数据原始 JSON，用 parseWafLog 解析 */
    extraData?: string;
  }

  /** CaddyLogReq 的受限查询参数；未传字段由后端默认值处理。 */
  export interface CaddyLogQuery {
    endTime?: string;
    host?: string;
    keyword?: string;
    /** 日志种类：access 只看访问日志，error 只看错误日志；不传则不过滤 */
    logKind?: 'access' | 'error';
    order?: 'asc' | 'desc';
    page?: number;
    pageSize?: number;
    sortBy?: 'logTime';
    startTime?: string;
    /** -1 表示不过滤状态码。 */
    status?: number;
    onlyWaf?: boolean;
  }

  export interface CaddyLogPageResult {
    list: CaddyLogItem[];
    total: number;
  }
}

/**
 * 获取 Caddy 配置 — GET /caddy/config
 */
export async function getCaddyConfigApi(config?: RequestClientConfig) {
  return requestClient.get<CaddyServerApi.CaddyConfig>('/caddy/config', config);
}

/**
 * 推送 Caddy 原始配置 — POST /caddy/config
 */
export async function pushCaddyConfigApi(
  config: string,
  modules?: string,
  baseHash?: string,
) {
  return requestClient.post<void>('/caddy/config', {
    baseHash,
    config,
    mode: modules ? 'quick' : 'raw',
    modules,
  } satisfies CaddyServerApi.CaddyConfigWrite);
}

/**
 * 预览 Caddy 配置变更 — POST /caddy/config/preview
 */
export async function previewCaddyConfigApi(
  data: CaddyServerApi.CaddyConfigWrite,
) {
  return requestClient.post<CaddyServerApi.ConfigPreview>(
    '/caddy/config/preview',
    data,
  );
}

/**
 * 获取配置历史列表 — GET /caddy/config/history
 */
export async function getCaddyConfigHistoryListApi(
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddyServerApi.ConfigHistoryListResult>(
    '/caddy/config/history',
    config,
  );
}

/**
 * 获取配置历史详情 — GET /caddy/config/history/:historyId
 */
export async function getCaddyConfigHistoryDetailApi(
  historyId: number,
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddyServerApi.ConfigHistoryDetail>(
    `/caddy/config/history/${historyId}`,
    config,
  );
}

/**
 * 回滚 Caddy 配置 — POST /caddy/config/rollback
 */
export async function rollbackCaddyConfigApi(
  data: CaddyServerApi.RollbackParams,
) {
  return requestClient.post<void>('/caddy/config/rollback', data);
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

/**
 * 站点近窗指标项 — POST /caddy/logs/site-metrics
 * window=24h 契约：total / blocked 为新增可选字段（后端 getSiteMetrics 扩展），
 * 未上线前为 undefined，前端按 — 降级展示，不做本地推测。
 */
export interface SiteMetricsItem {
  host: string;
  count4xx: number;
  count5xx: number;
  /** 窗口内总请求数（无 status 过滤，聚合自 caddy_logs） */
  total?: number;
  /** 窗口内 WAF 拦截数（按站点聚合自 waf_events，仅统计被拦截请求） */
  blocked?: number;
}

export interface SiteMetricsListResult {
  list: SiteMetricsItem[];
}

export interface SiteMetricsParams {
  /** 主机列表，单次最多 50 */
  hosts: string[];
  /** 时间窗分钟数，默认 15；"window=24h" 即传 1440 */
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
