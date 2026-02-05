import { request } from '../request';

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
  recent: DashboardRecentItem[];
  range: DashboardRange;
}

export function fetchDashboardSummary(params?: {
  startTime?: string;
  endTime?: string;
  intervalSec?: number;
  topN?: number;
  recentLimit?: number;
}) {
  return request<DashboardSummaryResp>({
    url: '/api/dashboard/summary',
    params
  });
}
