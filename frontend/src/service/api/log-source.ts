import { request } from '../request';

export interface LogSourceItem {
    id: number;
    name: string;
    path: string;
    type: string;
    enabled: boolean;
    createdAt: string;
}

export interface LogSourceListResp {
    list: LogSourceItem[];
    total: number;
}

export function fetchLogSourceList(params: { page: number; pageSize: number }) {
    return request<LogSourceListResp>({
        url: '/api/source',
        params
    });
}

export function createLogSource(data: { name?: string; path: string; type?: string }) {
    return request<any>({
        url: '/api/source',
        method: 'post',
        data
    });
}

export function updateLogSource(id: number, data: { name?: string; path?: string; enabled: boolean }) {
    return request<any>({
        url: `/api/source/${id}`,
        method: 'put',
        data
    });
}

export function deleteLogSource(id: number) {
    return request<any>({
        url: `/api/source/${id}`,
        method: 'delete'
    });
}
