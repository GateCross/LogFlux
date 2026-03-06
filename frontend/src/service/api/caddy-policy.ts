import { request } from '../request';

export type WafPolicyEngineMode = 'on' | 'off' | 'detectiononly';
export type WafPolicyAuditEngine = 'off' | 'on' | 'relevantonly';
export type WafPolicyAuditLogFormat = 'json' | 'native';
export type WafPolicyCrsTemplate = 'low_fp' | 'balanced' | 'high_blocking' | 'custom';
export type WafPolicyRevisionStatus = 'draft' | 'published' | 'rolled_back';
export type WafPolicyScopeType = 'global' | 'site' | 'route';
export type WafPolicyRemoveType = 'id' | 'tag';

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
