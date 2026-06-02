/**
 * Service_API: WAF policy stats & false-positive feedback (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/caddy_waf_observe.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

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

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/caddy/waf/policy/stats - Fetch WAF policy statistics. */
export function fetchWafPolicyStats(params?: {
  policyId?: number;
  startTime?: string;
  endTime?: string;
  intervalSec?: number;
  topN?: number;
  host?: string;
  path?: string;
  method?: string;
}): Promise<FlatResponse<WafPolicyStatsResp>> {
  return request<WafPolicyStatsResp>({ url: '/api/caddy/waf/policy/stats', params });
}

/** GET /api/caddy/waf/policy/false-positive-feedback - Fetch false-positive feedback list. */
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
}): Promise<FlatResponse<WafPolicyFalsePositiveFeedbackListResp>> {
  return request<WafPolicyFalsePositiveFeedbackListResp>({
    url: '/api/caddy/waf/policy/false-positive-feedback',
    params,
  });
}

/** POST /api/caddy/waf/policy/false-positive-feedback - Create false-positive feedback. */
export function createWafPolicyFalsePositiveFeedback(
  data: WafPolicyFalsePositiveFeedbackPayload,
): Promise<FlatResponse<void>> {
  return request<void>({ url: '/api/caddy/waf/policy/false-positive-feedback', method: 'post', data });
}

/** PUT /api/caddy/waf/policy/false-positive-feedback/:id/status - Update feedback status. */
export function updateWafPolicyFalsePositiveFeedbackStatus(
  id: number,
  data: WafPolicyFalsePositiveFeedbackStatusUpdatePayload,
): Promise<FlatResponse<void>> {
  return request<void>({
    url: `/api/caddy/waf/policy/false-positive-feedback/${id}/status`,
    method: 'put',
    data,
  });
}

/** PUT /api/caddy/waf/policy/false-positive-feedback/batch-status - Batch update feedback status. */
export function batchUpdateWafPolicyFalsePositiveFeedbackStatus(
  data: WafPolicyFalsePositiveFeedbackBatchStatusUpdatePayload,
): Promise<FlatResponse<WafPolicyFalsePositiveFeedbackBatchStatusUpdateResp>> {
  return request<WafPolicyFalsePositiveFeedbackBatchStatusUpdateResp>({
    url: '/api/caddy/waf/policy/false-positive-feedback/batch-status',
    method: 'put',
    data,
  });
}
