import { requestClient } from '#/api/request';

export namespace CaddyWafIntegrationApi {
  /** WAF 集成状态 — 对齐 backend WafIntegrationStatusResp。 */
  export interface IntegrationStatus {
    availableSites: string[];
    directiveReady: boolean;
    importedSites: string[];
    integrated: boolean;
    message?: string;
    orderReady: boolean;
    serverId: number;
    snippetReady: boolean;
  }

  /** WAF 集成应用请求 — 对齐 backend WafIntegrationApplyReq。 */
  export interface IntegrationApplyPayload {
    applyAll?: boolean;
    dryRun?: boolean;
    enabled: boolean;
    serverId?: number;
    siteAddresses?: string[];
  }

  /** WAF 集成应用结果 — 对齐 backend WafIntegrationApplyResp。 */
  export interface IntegrationApplyResult {
    actions: string[];
    changed: boolean;
    config?: string;
    enabled: boolean;
    importedSites: string[];
    message?: string;
    serverId: number;
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
export async function applyWafIntegrationApiWithPayload(
  data: CaddyWafIntegrationApi.IntegrationApplyPayload,
) {
  return requestClient.post<CaddyWafIntegrationApi.IntegrationApplyResult>(
    '/caddy/waf/integration/apply',
    data,
  );
}
