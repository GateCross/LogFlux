import { requestClient } from '#/api/request';

export namespace SystemLogApi {
  export interface LogItem {
    [key: string]: any;
  }
  export interface LogQuery {
    [key: string]: any;
  }
}

/**
 * 查询系统日志 — GET /system/logs
 */
export async function getSystemLogsApi(params?: SystemLogApi.LogQuery) {
  return requestClient.get<SystemLogApi.LogItem[]>('/system/logs', { params });
}
