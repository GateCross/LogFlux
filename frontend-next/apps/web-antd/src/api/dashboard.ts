import { requestClient } from '#/api/request';

export namespace DashboardApi {
  export interface Summary {
    // LogFlux dashboard summary data
    [key: string]: any;
  }
}

/**
 * 获取仪表盘汇总数据 — GET /dashboard/summary
 */
export async function getDashboardSummaryApi() {
  return requestClient.get<DashboardApi.Summary>('/dashboard/summary');
}
