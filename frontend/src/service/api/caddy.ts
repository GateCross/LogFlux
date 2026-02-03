import { request } from '../request';

export function fetchCaddyServers() {
    return request<any>({ url: '/api/caddy/server' });
}

export function addCaddyServer(data: any) {
    return request<any>({ url: '/api/caddy/server', method: 'post', data });
}

export function updateCaddyServer(data: any) {
    return request<any>({ url: `/api/caddy/server/${data.id}`, method: 'put', data });
}

export function deleteCaddyServer(id: number) {
    return request<any>({ url: `/api/caddy/server/${id}`, method: 'delete' });
}

export function fetchCaddyConfig(serverId: number) {
    return request<any>({ url: `/api/caddy/server/${serverId}/config` });
}

export function updateCaddyConfig(serverId: number, config: string, modules?: string) {
    return request<any>({
        url: `/api/caddy/server/${serverId}/config`,
        method: 'post',
        data: { config, modules }
    });
}

export function fetchCaddyLogs(params: { page: number; pageSize: number; keyword?: string; host?: string; status?: number; startTime?: string; endTime?: string }) {
    return request<any>({ url: '/api/caddy/logs', params });
}

export function fetchCaddyConfigHistory(serverId: number, params: { page: number; pageSize: number }) {
    return request<any>({ url: `/api/caddy/server/${serverId}/config/history`, params });
}

export function rollbackCaddyConfig(serverId: number, historyId: number) {
    return request<any>({
        url: `/api/caddy/server/${serverId}/config/rollback`,
        method: 'post',
        data: { historyId }
    });
}
