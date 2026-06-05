import { requestClient } from '#/api/request';

export namespace CaddySimpleWafApi {
  export type SimpleWafAudit = 'off' | 'on' | 'relevantonly';
  export type SimpleWafMode = 'detectiononly' | 'off' | 'on';
  export type SimpleWafStrength = 'balanced' | 'high_blocking' | 'low_fp';

  export interface SimpleWafConfig {
    [key: string]: any;
  }

  export interface SimpleWafConfigPayload {
    audit: SimpleWafAudit;
    enabled: boolean;
    mode: SimpleWafMode;
    requestBodyAccess: boolean;
    requestBodyLimit: number;
    requestBodyNoFilesLimit: number;
    serverId?: number;
    siteAddresses?: string[];
    strength: SimpleWafStrength;
  }
}

/**
 * 获取简易 WAF 配置 — GET /caddy/waf/simple-config
 */
export async function getSimpleWafConfigApi(serverId?: number) {
  return requestClient.get<CaddySimpleWafApi.SimpleWafConfig>(
    '/caddy/waf/simple-config',
    { params: serverId ? { serverId } : undefined },
  );
}

/**
 * 更新简易 WAF 配置 — PUT /caddy/waf/simple-config
 */
export async function updateSimpleWafConfigApi(
  data: CaddySimpleWafApi.SimpleWafConfigPayload,
) {
  return requestClient.put<void>('/caddy/waf/simple-config', data);
}

/**
 * 预览简易 WAF 配置 — POST /caddy/waf/simple-config/preview
 */
export async function previewSimpleWafConfigApi(
  data: CaddySimpleWafApi.SimpleWafConfigPayload,
) {
  return requestClient.post<CaddySimpleWafApi.SimpleWafConfig>(
    '/caddy/waf/simple-config/preview',
    data,
  );
}

/**
 * 应用简易 WAF 配置 — POST /caddy/waf/simple-config/apply
 */
export async function applySimpleWafConfigApi(
  data: CaddySimpleWafApi.SimpleWafConfigPayload,
) {
  return requestClient.post<CaddySimpleWafApi.SimpleWafConfig>(
    '/caddy/waf/simple-config/apply',
    data,
  );
}
