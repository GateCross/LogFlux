import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

export namespace CaddyWafEventApi {
  /** WAF 事件项 — 对齐 backend WafEventItem */
  export interface WafEvent {
    id: number;
    eventTime: string;
    transactionId: string;
    ruleId: string;
    ruleMsg: string;
    severity: string;
    interrupted: boolean;
    status: number;
    clientIp: string;
    country: string;
    province: string;
    city: string;
    host: string;
    method: string;
    uri: string;
    userAgent: string;
  }

  export interface WafEventListParams {
    page?: number;
    pageSize?: number;
    startTime?: string;
    endTime?: string;
    clientIp?: string;
    host?: string;
    ruleId?: string;
    severity?: string;
    /** block | detect | 空 = 全部 */
    action?: string;
    keyword?: string;
  }

  export interface WafEventListResult {
    list: WafEvent[];
    total: number;
  }

  export interface WafEventStatsParams {
    startTime?: string;
    endTime?: string;
    intervalSec?: number;
    topN?: number;
  }

  export interface WafEventTrendItem {
    time: string;
    total: number;
    blocked: number;
  }

  export interface WafEventCountItem {
    name: string;
    count: number;
  }

  export interface WafEventGeoItem {
    name: string;
    value: number;
  }

  /** WAF 安全统计 — 对齐 backend WafEventStatsResp */
  export interface WafEventStats {
    total: number;
    blocked: number;
    uniqueIps: number;
    trend: WafEventTrendItem[];
    topRules: WafEventCountItem[];
    topIps: WafEventCountItem[];
    topHosts: WafEventCountItem[];
    countries: WafEventGeoItem[];
  }
}

/**
 * 获取 WAF 事件列表 — GET /caddy/waf/events
 */
export async function getWafEventListApi(
  params?: CaddyWafEventApi.WafEventListParams,
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddyWafEventApi.WafEventListResult>(
    '/caddy/waf/events',
    { params, ...config },
  );
}

/**
 * 获取 WAF 安全统计 — GET /caddy/waf/events/stats
 */
export async function getWafEventStatsApi(
  params?: CaddyWafEventApi.WafEventStatsParams,
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddyWafEventApi.WafEventStats>(
    '/caddy/waf/events/stats',
    { params, ...config },
  );
}
