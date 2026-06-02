/**
 * Service_API: IP region config (migrated from old Vue frontend).
 *
 * Endpoints align with `backend/api/caddy_ip_region.api`.
 * All calls go through the unified Request_Layer (`@/utils/request`),
 * returning flat `{ data, error }`.
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

// ─── Types ───────────────────────────────────────────────────────────────────

export interface IPRegionConfig {
  enabled: boolean;
  allowList: string[];
}

// ─── Functions ───────────────────────────────────────────────────────────────

/** GET /api/caddy/ip-region - Fetch IP region config. */
export function fetchIPRegionConfig(): Promise<FlatResponse<IPRegionConfig>> {
  return request<IPRegionConfig>({ url: '/api/caddy/ip-region' });
}

/** PUT /api/caddy/ip-region - Update IP region config. */
export function updateIPRegionConfig(data: IPRegionConfig): Promise<FlatResponse<IPRegionConfig>> {
  return request<IPRegionConfig>({ url: '/api/caddy/ip-region', method: 'put', data });
}
