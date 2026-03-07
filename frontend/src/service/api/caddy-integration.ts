import { request } from '../request';

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

export function fetchWafIntegrationStatus() {
  return request<WafIntegrationStatusResp>({ url: '/api/caddy/waf/integration/status' });
}

export function applyWafIntegration(data: WafIntegrationApplyPayload) {
  return request<WafIntegrationApplyResp>({
    url: '/api/caddy/waf/integration/apply',
    method: 'post',
    data
  });
}
