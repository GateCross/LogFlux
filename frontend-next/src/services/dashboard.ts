/**
 * Service_API：仪表盘模块（对齐 backend/api/dashboard.api，Req 17.3 / 8.1）。
 *
 * 迁移自旧 Vue 版 `frontend/src/service/api/dashboard.ts`，统一经 Request_Layer
 * （`@/utils/request`）返回扁平 `{ data, error }`。
 */
import { request } from '@/utils/request';

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

/** GET /api/dashboard/summary —— 加载仪表盘汇总数据（Req 8.1）。 */
export function fetchDashboardSummary(params?: {
  startTime?: string;
  endTime?: string;
  intervalSec?: number;
  topN?: number;
  recentLimit?: number;
}) {
  return request<DashboardSummaryResp>({
    url: '/api/dashboard/summary',
    params,
  });
}
