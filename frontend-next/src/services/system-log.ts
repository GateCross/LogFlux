/**
 * Service_API：系统日志模块（对齐 backend/api/system_log.api，Req 17.3）。
 *
 * 迁移自旧 Vue 版 `frontend/src/service/api/system-log.ts`，统一经 Request_Layer
 * 返回扁平 `{ data, error }`。
 */
import { request } from '@/utils/request';

export interface SystemLogItem {
  id: number;
  logTime: string;
  level: string;
  message: string;
  caller: string;
  traceId?: string;
  spanId?: string;
  source: string;
  rawLog: string;
  extraData: string;
}

export interface SystemLogResp {
  list: SystemLogItem[];
  total: number;
}

/** GET /api/system/logs —— 分页查询系统日志。 */
export function fetchSystemLogs(params: {
  page: number;
  pageSize: number;
  keyword?: string;
  source?: string;
  level?: string;
  startTime?: string;
  endTime?: string;
  sortBy?: string;
  order?: string;
}) {
  return request<SystemLogResp>({
    url: '/api/system/logs',
    params,
  });
}
