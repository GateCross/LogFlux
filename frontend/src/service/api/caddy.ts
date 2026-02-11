import { request } from '../request';

export type WafKind = 'crs' | 'coraza_engine';
export type WafMode = 'remote' | 'manual';
export type WafAuthType = 'none' | 'token' | 'basic';
export type WafReleaseStatus = 'downloaded' | 'verified' | 'active' | 'failed' | 'rolled_back';
export type WafJobStatus = 'running' | 'success' | 'failed';

export interface WafSourceItem {
    id: number;
    name: string;
    kind: WafKind;
    mode: WafMode;
    url: string;
    checksumUrl: string;
    proxyUrl?: string;
    authType: WafAuthType;
    schedule: string;
    enabled: boolean;
    autoCheck: boolean;
    autoDownload: boolean;
    autoActivate: boolean;
    lastRelease: string;
    lastError: string;
    createdAt: string;
    updatedAt: string;
}

export interface WafSourceListResp {
    list: WafSourceItem[];
    total: number;
}

export interface WafSourcePayload {
    name: string;
    kind?: WafKind;
    mode?: WafMode;
    url?: string;
    checksumUrl?: string;
    proxyUrl?: string;
    authType?: WafAuthType;
    authSecret?: string;
    schedule?: string;
    enabled?: boolean;
    autoCheck?: boolean;
    autoDownload?: boolean;
    autoActivate?: boolean;
    meta?: string;
}

export interface WafReleaseItem {
    id: number;
    sourceId: number;
    kind: WafKind;
    version: string;
    artifactType: string;
    checksum: string;
    sizeBytes: number;
    storagePath: string;
    status: WafReleaseStatus;
    createdAt: string;
    updatedAt: string;
}

export interface WafReleaseListResp {
    list: WafReleaseItem[];
    total: number;
}

export interface WafJobItem {
    id: number;
    sourceId: number;
    releaseId: number;
    action: string;
    triggerMode: string;
    operator: string;
    status: WafJobStatus;
    message: string;
    startedAt: string;
    finishedAt: string;
    createdAt: string;
}

export interface WafJobListResp {
    list: WafJobItem[];
    total: number;
}

export interface WafEngineStatusResp {
    currentVersion?: string;
    latestVersion?: string;
    canUpgrade?: boolean;
    checkedAt?: string;
    source?: string;
    message?: string;
    [key: string]: unknown;
}

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

export function updateCaddyConfigRaw(serverId: number, config: string) {
    return request<any>({
        url: `/api/caddy/server/${serverId}/config`,
        method: 'post',
        data: { config }
    });
}

export function updateCaddyConfigStructured(serverId: number, config: string, modules: string) {
    return request<any>({
        url: `/api/caddy/server/${serverId}/config`,
        method: 'post',
        data: { config, modules }
    });
}

export function fetchCaddyLogs(params: { page: number; pageSize: number; keyword?: string; host?: string; status?: number; startTime?: string; endTime?: string; sortBy?: string; order?: string }) {
    return request<any>({ url: '/api/caddy/logs', params });
}

export function fetchCaddyConfigHistory(serverId: number, params: { page: number; pageSize: number }) {
    return request<any>({ url: `/api/caddy/server/${serverId}/config/history`, params });
}

export function fetchCaddyConfigHistoryDetail(serverId: number, historyId: number) {
    return request<any>({ url: `/api/caddy/server/${serverId}/config/history/${historyId}` });
}

export function rollbackCaddyConfig(serverId: number, historyId: number) {
    return request<any>({
        url: `/api/caddy/server/${serverId}/config/rollback`,
        method: 'post',
        data: { historyId }
    });
}

export function fetchWafSourceList(params: { page: number; pageSize: number; kind?: WafKind | ''; name?: string }) {
    return request<WafSourceListResp>({ url: '/api/caddy/waf/source', params });
}

export function createWafSource(data: WafSourcePayload) {
    return request<any>({ url: '/api/caddy/waf/source', method: 'post', data });
}

export function updateWafSource(id: number, data: Partial<WafSourcePayload>) {
    return request<any>({ url: `/api/caddy/waf/source/${id}`, method: 'put', data });
}

export function deleteWafSource(id: number) {
    return request<any>({ url: `/api/caddy/waf/source/${id}`, method: 'delete' });
}

export function checkWafSource(id: number) {
    return request<any>({ url: `/api/caddy/waf/source/${id}/check`, method: 'post' });
}

export function syncWafSource(id: number, activateNow?: boolean) {
    return request<any>({
        url: `/api/caddy/waf/source/${id}/sync`,
        method: 'post',
        timeout: 240000,
        data: { activateNow }
    });
}

export function uploadWafPackage(data: FormData) {
    return request<any>({
        url: '/api/caddy/waf/upload',
        method: 'post',
        data
    });
}

export function fetchWafReleaseList(params: {
    page: number;
    pageSize: number;
    kind?: WafKind | '';
    status?: WafReleaseStatus | '';
}) {
    return request<WafReleaseListResp>({ url: '/api/caddy/waf/release', params });
}

export function activateWafRelease(id: number) {
    return request<any>({ url: `/api/caddy/waf/release/${id}/activate`, method: 'post' });
}

export function rollbackWafRelease(data: { target?: 'last_good' | 'version'; version?: string }) {
    return request<any>({
        url: '/api/caddy/waf/release/rollback',
        method: 'post',
        data
    });
}

export function fetchWafJobList(params: { page: number; pageSize: number; status?: WafJobStatus | ''; action?: string }) {
    return request<WafJobListResp>({ url: '/api/caddy/waf/job', params });
}

export function fetchWafEngineStatus() {
    return request<WafEngineStatusResp>({ url: '/api/caddy/waf/engine/status' });
}

export function checkWafEngine() {
    return request<any>({ url: '/api/caddy/waf/engine/check', method: 'post' });
}
