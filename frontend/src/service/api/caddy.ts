import { request } from '../request';

export type WafKind = 'crs' | 'coraza_engine';
export type WafMode = 'remote' | 'manual';
export type WafAuthType = 'none' | 'token' | 'basic';
export type WafReleaseStatus = 'downloaded' | 'verified' | 'active' | 'failed' | 'rolled_back';
export type WafJobStatus = 'running' | 'success' | 'failed';
export type WafPolicyEngineMode = 'on' | 'off' | 'detectiononly';
export type WafPolicyAuditEngine = 'off' | 'on' | 'relevantonly';
export type WafPolicyAuditLogFormat = 'json' | 'native';
export type WafPolicyCrsTemplate = 'low_fp' | 'balanced' | 'high_blocking' | 'custom';
export type WafPolicyRevisionStatus = 'draft' | 'published' | 'rolled_back';
export type WafPolicyScopeType = 'global' | 'site' | 'route';
export type WafPolicyRemoveType = 'id' | 'tag';

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

export interface WafPolicyItem {
    id: number;
    name: string;
    description: string;
    enabled: boolean;
    isDefault: boolean;
    engineMode: WafPolicyEngineMode;
    auditEngine: WafPolicyAuditEngine;
    auditLogFormat: WafPolicyAuditLogFormat;
    auditRelevantStatus: string;
    requestBodyAccess: boolean;
    requestBodyLimit: number;
    requestBodyNoFilesLimit: number;
    crsTemplate: WafPolicyCrsTemplate;
    crsParanoiaLevel: number;
    crsInboundAnomalyThreshold: number;
    crsOutboundAnomalyThreshold: number;
    config: string;
    createdAt: string;
    updatedAt: string;
}

export interface WafPolicyListResp {
    list: WafPolicyItem[];
    total: number;
}

export interface WafPolicyPayload {
    name: string;
    description?: string;
    enabled?: boolean;
    isDefault?: boolean;
    engineMode?: WafPolicyEngineMode;
    auditEngine?: WafPolicyAuditEngine;
    auditLogFormat?: WafPolicyAuditLogFormat;
    auditRelevantStatus?: string;
    requestBodyAccess?: boolean;
    requestBodyLimit?: number;
    requestBodyNoFilesLimit?: number;
    crsTemplate?: WafPolicyCrsTemplate;
    crsParanoiaLevel?: number;
    crsInboundAnomalyThreshold?: number;
    crsOutboundAnomalyThreshold?: number;
    config?: string;
}

export interface WafPolicyRevisionItem {
    id: number;
    policyId: number;
    policyName: string;
    version: number;
    status: WafPolicyRevisionStatus;
    operator: string;
    message: string;
    changeSummary: string;
    createdAt: string;
    updatedAt: string;
}

export interface WafPolicyRevisionListResp {
    list: WafPolicyRevisionItem[];
    total: number;
}

export interface WafPolicyPreviewResp {
    directives: string;
}

export interface WafPolicyStatsItem {
    policyId: number;
    policyName: string;
    hitCount: number;
    blockedCount: number;
    allowedCount: number;
    suspectedFalsePositiveCount: number;
    blockRate: number;
}

export interface WafPolicyStatsTrendItem {
    time: string;
    hitCount: number;
    blockedCount: number;
    allowedCount: number;
}

export interface WafPolicyStatsDimensionItem {
    key: string;
    hitCount: number;
    blockedCount: number;
    allowedCount: number;
    blockRate: number;
}

export interface WafPolicyStatsResp {
    range: {
        startTime: string;
        endTime: string;
        intervalSec: number;
    };
    summary: WafPolicyStatsItem;
    list: WafPolicyStatsItem[];
    trend: WafPolicyStatsTrendItem[];
    topHosts: WafPolicyStatsDimensionItem[];
    topPaths: WafPolicyStatsDimensionItem[];
    topMethods: WafPolicyStatsDimensionItem[];
}

export interface WafPolicyFalsePositiveFeedbackItem {
    id: number;
    policyId: number;
    policyName: string;
    host: string;
    path: string;
    method: string;
    status: number;
    feedbackStatus: 'pending' | 'confirmed' | 'resolved';
    assignee: string;
    dueAt: string;
    isOverdue: boolean;
    sampleUri: string;
    reason: string;
    suggestion: string;
    operator: string;
    processNote: string;
    processedBy: string;
    processedAt: string;
    createdAt: string;
}

export interface WafPolicyFalsePositiveFeedbackListResp {
    list: WafPolicyFalsePositiveFeedbackItem[];
    total: number;
}

export interface WafPolicyFalsePositiveFeedbackPayload {
    policyId?: number;
    host?: string;
    path?: string;
    method?: string;
    status?: number;
    assignee?: string;
    dueAt?: string;
    sampleUri?: string;
    reason: string;
    suggestion?: string;
}

export interface WafPolicyFalsePositiveFeedbackStatusUpdatePayload {
    feedbackStatus: 'pending' | 'confirmed' | 'resolved';
    processNote?: string;
    assignee?: string;
    dueAt?: string;
}

export interface WafPolicyFalsePositiveFeedbackBatchStatusUpdatePayload {
    ids: number[];
    feedbackStatus: 'pending' | 'confirmed' | 'resolved';
    processNote?: string;
    assignee?: string;
    dueAt?: string;
}

export interface WafPolicyFalsePositiveFeedbackBatchStatusUpdateResp {
    affectedCount: number;
    processedBy: string;
    processedAt: string;
}

export interface WafRuleExclusionItem {
    id: number;
    policyId: number;
    name: string;
    description: string;
    enabled: boolean;
    scopeType: WafPolicyScopeType;
    host: string;
    path: string;
    method: string;
    removeType: WafPolicyRemoveType;
    removeValue: string;
    createdAt: string;
    updatedAt: string;
}

export interface WafRuleExclusionListResp {
    list: WafRuleExclusionItem[];
    total: number;
}

export interface WafRuleExclusionPayload {
    policyId: number;
    name?: string;
    description?: string;
    enabled?: boolean;
    scopeType?: WafPolicyScopeType;
    host?: string;
    path?: string;
    method?: string;
    removeType?: WafPolicyRemoveType;
    removeValue: string;
}

export interface WafPolicyBindingItem {
    id: number;
    policyId: number;
    name: string;
    description: string;
    enabled: boolean;
    scopeType: WafPolicyScopeType;
    host: string;
    path: string;
    method: string;
    priority: number;
    createdAt: string;
    updatedAt: string;
}

export interface WafPolicyBindingListResp {
    list: WafPolicyBindingItem[];
    total: number;
}

export interface WafPolicyBindingPayload {
    policyId: number;
    name?: string;
    description?: string;
    enabled?: boolean;
    scopeType?: WafPolicyScopeType;
    host?: string;
    path?: string;
    method?: string;
    priority?: number;
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

export function clearWafReleases(data?: { kind?: WafKind | '' }) {
    return request<any>({
        url: '/api/caddy/waf/release/clear',
        method: 'post',
        data
    });
}

export function fetchWafJobList(params: { page: number; pageSize: number; status?: WafJobStatus | ''; action?: string }) {
    return request<WafJobListResp>({ url: '/api/caddy/waf/job', params });
}

export function clearWafJobs() {
    return request<any>({
        url: '/api/caddy/waf/job/clear',
        method: 'post'
    });
}

export function fetchWafEngineStatus() {
    return request<WafEngineStatusResp>({ url: '/api/caddy/waf/engine/status' });
}

export function checkWafEngine() {
    return request<any>({ url: '/api/caddy/waf/engine/check', method: 'post' });
}

export function fetchWafPolicyList(params: { page: number; pageSize: number; name?: string }) {
    return request<WafPolicyListResp>({ url: '/api/caddy/waf/policy', params });
}

export function createWafPolicy(data: WafPolicyPayload) {
    return request<any>({ url: '/api/caddy/waf/policy', method: 'post', data });
}

export function updateWafPolicy(id: number, data: Partial<WafPolicyPayload>) {
    return request<any>({ url: `/api/caddy/waf/policy/${id}`, method: 'put', data });
}

export function deleteWafPolicy(id: number) {
    return request<any>({ url: `/api/caddy/waf/policy/${id}`, method: 'delete' });
}

export function previewWafPolicy(id: number) {
    return request<WafPolicyPreviewResp>({ url: `/api/caddy/waf/policy/${id}/preview`, method: 'post' });
}

export function validateWafPolicy(id: number) {
    return request<any>({ url: `/api/caddy/waf/policy/${id}/validate`, method: 'post' });
}

export function publishWafPolicy(id: number) {
    return request<any>({ url: `/api/caddy/waf/policy/${id}/publish`, method: 'post' });
}

export function rollbackWafPolicy(data: { revisionId: number }) {
    return request<any>({
        url: '/api/caddy/waf/policy/rollback',
        method: 'post',
        data
    });
}

export function fetchWafPolicyRevisionList(params: { page: number; pageSize: number; policyId?: number }) {
    return request<WafPolicyRevisionListResp>({ url: '/api/caddy/waf/policy/revision', params });
}

export function fetchWafPolicyStats(params?: {
    policyId?: number;
    startTime?: string;
    endTime?: string;
    intervalSec?: number;
    topN?: number;
    host?: string;
    path?: string;
    method?: string;
}) {
    return request<WafPolicyStatsResp>({ url: '/api/caddy/waf/policy/stats', params });
}

export function fetchWafPolicyFalsePositiveFeedbackList(params: {
    page: number;
    pageSize: number;
    policyId?: number;
    host?: string;
    path?: string;
    method?: string;
    feedbackStatus?: 'pending' | 'confirmed' | 'resolved';
    assignee?: string;
    slaStatus?: 'all' | 'normal' | 'overdue' | 'resolved';
}) {
    return request<WafPolicyFalsePositiveFeedbackListResp>({ url: '/api/caddy/waf/policy/false-positive-feedback', params });
}

export function createWafPolicyFalsePositiveFeedback(data: WafPolicyFalsePositiveFeedbackPayload) {
    return request<any>({ url: '/api/caddy/waf/policy/false-positive-feedback', method: 'post', data });
}

export function updateWafPolicyFalsePositiveFeedbackStatus(id: number, data: WafPolicyFalsePositiveFeedbackStatusUpdatePayload) {
    return request<any>({ url: `/api/caddy/waf/policy/false-positive-feedback/${id}/status`, method: 'put', data });
}

export function batchUpdateWafPolicyFalsePositiveFeedbackStatus(data: WafPolicyFalsePositiveFeedbackBatchStatusUpdatePayload) {
    return request<WafPolicyFalsePositiveFeedbackBatchStatusUpdateResp>({
        url: '/api/caddy/waf/policy/false-positive-feedback/batch-status',
        method: 'put',
        data
    });
}

export function fetchWafRuleExclusionList(params: {
    page: number;
    pageSize: number;
    policyId?: number;
    scopeType?: WafPolicyScopeType | '';
    name?: string;
}) {
    return request<WafRuleExclusionListResp>({ url: '/api/caddy/waf/policy/exclusion', params });
}

export function createWafRuleExclusion(data: WafRuleExclusionPayload) {
    return request<any>({ url: '/api/caddy/waf/policy/exclusion', method: 'post', data });
}

export function updateWafRuleExclusion(id: number, data: WafRuleExclusionPayload) {
    return request<any>({ url: `/api/caddy/waf/policy/exclusion/${id}`, method: 'put', data });
}

export function deleteWafRuleExclusion(id: number) {
    return request<any>({ url: `/api/caddy/waf/policy/exclusion/${id}`, method: 'delete' });
}

export function fetchWafPolicyBindingList(params: {
    page: number;
    pageSize: number;
    policyId?: number;
    scopeType?: WafPolicyScopeType | '';
    name?: string;
}) {
    return request<WafPolicyBindingListResp>({ url: '/api/caddy/waf/policy/binding', params });
}

export function createWafPolicyBinding(data: WafPolicyBindingPayload) {
    return request<any>({ url: '/api/caddy/waf/policy/binding', method: 'post', data });
}

export function updateWafPolicyBinding(id: number, data: WafPolicyBindingPayload) {
    return request<any>({ url: `/api/caddy/waf/policy/binding/${id}`, method: 'put', data });
}

export function deleteWafPolicyBinding(id: number) {
    return request<any>({ url: `/api/caddy/waf/policy/binding/${id}`, method: 'delete' });
}
