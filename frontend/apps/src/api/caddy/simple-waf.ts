import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

export namespace CaddySimpleWafApi {
  export type SimpleWafAudit = 'off' | 'on' | 'relevantonly';
  export type SimpleWafMode = 'detectiononly' | 'off' | 'on';
  export type SimpleWafStrength = 'balanced' | 'high_blocking' | 'low_fp';

  export interface CrsRuleItem {
    file: string;
    category: string;
  }

  export interface CrsRuleSummary {
    total: number;
    categories: string[];
    rules: CrsRuleItem[];
  }

  /** 简易 WAF 配置读响应 — 对齐 backend SimpleWafConfigResp */
  export interface SimpleWafConfig {
    serverId: number;
    enabled: boolean;
    integrated?: boolean;
    mode: SimpleWafMode | string;
    strength: SimpleWafStrength | string;
    audit: SimpleWafAudit | string;
    requestBodyAccess: boolean;
    requestBodyLimit: number;
    requestBodyNoFilesLimit: number;
    siteAddresses: string[];
    availableSites: string[];
    /** Caddyfile 里实际 import waf_protect 的站点 */
    importedSites: string[];
    corazaVersion?: string;
    crsVersion?: string;
    actions?: string[];
    directives?: string;
    config?: string;
    message?: string;
    rulesInfo?: CrsRuleSummary;
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
    /** 编辑起点的 Caddyfile 哈希；与库内不一致时后端拒绝写入 */
    baseHash?: string;
  }
}

/**
 * 获取简易 WAF 配置 — GET /caddy/waf/simple-config
 */
export async function getSimpleWafConfigApi(
  serverId?: number,
  config?: RequestClientConfig,
) {
  return requestClient.get<CaddySimpleWafApi.SimpleWafConfig>(
    '/caddy/waf/simple-config',
    { params: serverId ? { serverId } : undefined, ...config },
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
