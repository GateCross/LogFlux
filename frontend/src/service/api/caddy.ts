import { request } from '../request';

export function fetchCaddyServers() {
    return request<any>({ url: '/caddy/server' });
}

export function addCaddyServer(data: any) {
    return request<any>({ url: '/caddy/server', method: 'post', data });
}

export function updateCaddyServer(data: any) {
    return request<any>({ url: `/caddy/server/${data.id}`, method: 'put', data });
}

export function deleteCaddyServer(id: number) {
    return request<any>({ url: `/caddy/server/${id}`, method: 'delete' });
}

export function fetchCaddyConfig(serverId: number) {
    return request<any>({ url: `/caddy/server/${serverId}/config` });
}

export function updateCaddyConfig(serverId: number, config: string) {
    return request<any>({ url: `/caddy/server/${serverId}/config`, method: 'post', data: { config } });
}
