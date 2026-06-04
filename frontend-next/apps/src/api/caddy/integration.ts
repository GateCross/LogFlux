import { requestClient } from '#/api/request';

export namespace CaddyWafIntegrationApi {
  export interface IntegrationStatus {
    [key: string]: any;
  }

  export interface IntegrationApplyPayload {
    applyAll?: boolean;
    dryRun?: boolean;
    enabled: boolean;
    serverId?: number;
    siteAddresses?: string[];
  }

  export interface IntegrationApplyResult {
    [key: string]: any;
  }
}

/**
 * 检查 WAF 集成状态 — GET /caddy/waf/integration/status
 */
export async function getWafIntegrationStatusApi() {
  return requestClient.get<CaddyWafIntegrationApi.IntegrationStatus>(
    '/caddy/waf/integration/status',
  );
}

/**
 * 应用 WAF 集成 — POST /caddy/waf/integration/apply
 */
export async function applyWafIntegrationApi() {
  return requestClient.post<void>('/caddy/waf/integration/apply');
}

export async function applyWafIntegrationApiWithPayload(
  data: CaddyWafIntegrationApi.IntegrationApplyPayload,
) {
  return requestClient.post<CaddyWafIntegrationApi.IntegrationApplyResult>(
    '/caddy/waf/integration/apply',
    data,
  );
}
