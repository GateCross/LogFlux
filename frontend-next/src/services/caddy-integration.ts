/**
 * Service_API: WAF integration (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/caddy_waf_integration.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

export interface WafIntegrationStatusResp {
  serverId: number;
  integrated: boolean;
  orderReady: boolean;
  snippetReady: boolean;
  directiveReady: boolean;
  importedSites: string[];
  availableSites: string[];
  message?: string;
}

export interface WafIntegrationApplyPayload {
  serverId?: number;
  enabled: boolean;
  applyAll?: boolean;
  siteAddresses?: string[];
  dryRun?: boolean;
}

export interface WafIntegrationApplyResp {
  serverId: number;
  enabled: boolean;
  changed: boolean;
  importedSites: string[];
  actions: string[];
  config?: string;
  message?: string;
}

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/caddy/waf/integration/status - Fetch WAF integration status. */
export function fetchWafIntegrationStatus(): Promise<FlatResponse<WafIntegrationStatusResp>> {
  return request<WafIntegrationStatusResp>({ url: '/api/caddy/waf/integration/status' });
}

/** POST /api/caddy/waf/integration/apply - Apply WAF integration. */
export function applyWafIntegration(
  data: WafIntegrationApplyPayload,
): Promise<FlatResponse<WafIntegrationApplyResp>> {
  return request<WafIntegrationApplyResp>({
    url: '/api/caddy/waf/integration/apply',
    method: 'post',
    data,
  });
}
