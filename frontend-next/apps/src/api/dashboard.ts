import { requestClient } from '#/api/request';

export interface DashboardStats {
  requests: number;
  pv: number;
  uv: number;
  uniqueIp: number;
  blocked: number;
  attackIp: number;
}

export interface DashboardErrorStats {
  error4xx: number;
  blocked4xx: number;
  error5xx: number;
}

export interface DashboardTrendItem {
  time: string;
  value: number;
}

export interface DashboardGeoItem {
  name: string;
  value: number;
}

export interface DashboardRecentItem {
  id: number;
  logTime: string;
  method: string;
  uri: string;
  status: number;
  remoteIp: string;
  country: string;
}

export interface DashboardRange {
  startTime: string;
  endTime: string;
  intervalSec: number;
}

export interface DashboardSummaryResp {
  stats: DashboardStats;
  errorStats: DashboardErrorStats;
  trend: DashboardTrendItem[];
  geo: DashboardGeoItem[];
  geoProvince: DashboardGeoItem[];
  recent: DashboardRecentItem[];
  range: DashboardRange;
}

/**
 * 获取仪表盘汇总数据 — GET /dashboard/summary
 */
export async function getDashboardSummaryApi(params?: {
  startTime?: string;
  endTime?: string;
  intervalSec?: number;
  topN?: number;
  recentLimit?: number;
}) {
  return requestClient.get<DashboardSummaryResp>('/dashboard/summary', {
    params,
  });
}
