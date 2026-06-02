/**
 * Service_API: WAF policy management (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/caddy_waf_policy.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

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

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/caddy/waf/policy - Fetch WAF policy list. */
export function fetchWafPolicyList(params: {
  page: number;
  pageSize: number;
  name?: string;
}): Promise<FlatResponse<WafPolicyListResp>> {
  return request<WafPolicyListResp>({ url: '/api/caddy/waf/policy', params });
}

/** POST /api/caddy/waf/policy - Create a new WAF policy. */
export function createWafPolicy(data: WafPolicyPayload): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/caddy/waf/policy', method: 'post', data });
}

/** PUT /api/caddy/waf/policy/:id - Update an existing WAF policy. */
export function updateWafPolicy(id: number, data: Partial<WafPolicyPayload>): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/policy/${id}`, method: 'put', data });
}

/** DELETE /api/caddy/waf/policy/:id - Delete a WAF policy. */
export function deleteWafPolicy(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/policy/${id}`, method: 'delete' });
}

/** POST /api/caddy/waf/policy/:id/preview - Preview policy directives. */
export function previewWafPolicy(id: number): Promise<FlatResponse<WafPolicyPreviewResp>> {
  return request<WafPolicyPreviewResp>({ url: `/api/caddy/waf/policy/${id}/preview`, method: 'post' });
}

/** POST /api/caddy/waf/policy/:id/validate - Validate a WAF policy. */
export function validateWafPolicy(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/policy/${id}/validate`, method: 'post' });
}

/** POST /api/caddy/waf/policy/:id/publish - Publish a WAF policy. */
export function publishWafPolicy(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/policy/${id}/publish`, method: 'post' });
}

/** POST /api/caddy/waf/policy/rollback - Rollback a WAF policy to a revision. */
export function rollbackWafPolicy(data: { revisionId: number }): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/caddy/waf/policy/rollback',
    method: 'post',
    data,
  });
}

/** GET /api/caddy/waf/policy/revision - Fetch WAF policy revision list. */
export function fetchWafPolicyRevisionList(params: {
  page: number;
  pageSize: number;
  policyId?: number;
}): Promise<FlatResponse<WafPolicyRevisionListResp>> {
  return request<WafPolicyRevisionListResp>({ url: '/api/caddy/waf/policy/revision', params });
}

/** GET /api/caddy/waf/policy/exclusion - Fetch WAF rule exclusion list. */
export function fetchWafRuleExclusionList(params: {
  page: number;
  pageSize: number;
  policyId?: number;
  scopeType?: WafPolicyScopeType | '';
  name?: string;
}): Promise<FlatResponse<WafRuleExclusionListResp>> {
  return request<WafRuleExclusionListResp>({ url: '/api/caddy/waf/policy/exclusion', params });
}

/** POST /api/caddy/waf/policy/exclusion - Create a WAF rule exclusion. */
export function createWafRuleExclusion(data: WafRuleExclusionPayload): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/caddy/waf/policy/exclusion', method: 'post', data });
}

/** PUT /api/caddy/waf/policy/exclusion/:id - Update a WAF rule exclusion. */
export function updateWafRuleExclusion(id: number, data: WafRuleExclusionPayload): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/policy/exclusion/${id}`, method: 'put', data });
}

/** DELETE /api/caddy/waf/policy/exclusion/:id - Delete a WAF rule exclusion. */
export function deleteWafRuleExclusion(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/policy/exclusion/${id}`, method: 'delete' });
}

/** GET /api/caddy/waf/policy/binding - Fetch WAF policy binding list. */
export function fetchWafPolicyBindingList(params: {
  page: number;
  pageSize: number;
  policyId?: number;
  scopeType?: WafPolicyScopeType | '';
  name?: string;
}): Promise<FlatResponse<WafPolicyBindingListResp>> {
  return request<WafPolicyBindingListResp>({ url: '/api/caddy/waf/policy/binding', params });
}

/** POST /api/caddy/waf/policy/binding - Create a WAF policy binding. */
export function createWafPolicyBinding(data: WafPolicyBindingPayload): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/caddy/waf/policy/binding', method: 'post', data });
}

/** PUT /api/caddy/waf/policy/binding/:id - Update a WAF policy binding. */
export function updateWafPolicyBinding(id: number, data: WafPolicyBindingPayload): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/policy/binding/${id}`, method: 'put', data });
}

/** DELETE /api/caddy/waf/policy/binding/:id - Delete a WAF policy binding. */
export function deleteWafPolicyBinding(id: number): Promise<FlatResponse<void>> {
  return request<void>({ url: `/api/caddy/waf/policy/binding/${id}`, method: 'delete' });
}
