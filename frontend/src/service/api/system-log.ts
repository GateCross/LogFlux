import { request } from '../request';

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
        params
    });
}
