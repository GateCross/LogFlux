/**
 * Service_API: Simple WAF config (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/caddy_waf_simple_config.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

export type SimpleWafMode = 'off' | 'detectiononly' | 'on';
export type SimpleWafStrength = 'low_fp' | 'balanced' | 'high_blocking';
export type SimpleWafAudit = 'off' | 'relevantonly' | 'on';

export interface SimpleWafConfigResp {
  serverId: number;
  enabled: boolean;
  integrated: boolean;
  mode: SimpleWafMode;
  strength: SimpleWafStrength;
  audit: SimpleWafAudit;
  requestBodyAccess: boolean;
  requestBodyLimit: number;
  requestBodyNoFilesLimit: number;
  siteAddresses: string[];
  availableSites: string[];
  corazaVersion?: string;
  crsVersion?: string;
  actions?: string[];
  directives?: string;
  config?: string;
  message?: string;
}

export interface SimpleWafConfigPayload {
  serverId?: number;
  enabled: boolean;
  mode: SimpleWafMode;
  strength: SimpleWafStrength;
  audit: SimpleWafAudit;
  requestBodyAccess: boolean;
  requestBodyLimit: number;
  requestBodyNoFilesLimit: number;
  siteAddresses?: string[];
}

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/caddy/waf/simple-config - Fetch simple WAF config. */
export function fetchSimpleWafConfig(serverId?: number): Promise<FlatResponse<SimpleWafConfigResp>> {
  return request<SimpleWafConfigResp>({
    url: '/api/caddy/waf/simple-config',
    params: serverId ? { serverId } : undefined,
  });
}

/** PUT /api/caddy/waf/simple-config - Update simple WAF config. */
export function updateSimpleWafConfig(data: SimpleWafConfigPayload): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/caddy/waf/simple-config',
    method: 'put',
    data,
  });
}

/** POST /api/caddy/waf/simple-config/preview - Preview simple WAF config changes. */
export function previewSimpleWafConfig(data: SimpleWafConfigPayload): Promise<FlatResponse<SimpleWafConfigResp>> {
  return request<SimpleWafConfigResp>({
    url: '/api/caddy/waf/simple-config/preview',
    method: 'post',
    data,
  });
}

/** POST /api/caddy/waf/simple-config/apply - Apply simple WAF config. */
export function applySimpleWafConfig(data: SimpleWafConfigPayload): Promise<FlatResponse<SimpleWafConfigResp>> {
  return request<SimpleWafConfigResp>({
    url: '/api/caddy/waf/simple-config/apply',
    method: 'post',
    data,
  });
}
