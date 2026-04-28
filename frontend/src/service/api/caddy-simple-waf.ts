import { request } from '../request';

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

export function fetchSimpleWafConfig(serverId?: number) {
  return request<SimpleWafConfigResp>({
    url: '/api/caddy/waf/simple-config',
    params: serverId ? { serverId } : undefined
  });
}

export function updateSimpleWafConfig(data: SimpleWafConfigPayload) {
  return request<any>({
    url: '/api/caddy/waf/simple-config',
    method: 'put',
    data
  });
}

export function previewSimpleWafConfig(data: SimpleWafConfigPayload) {
  return request<SimpleWafConfigResp>({
    url: '/api/caddy/waf/simple-config/preview',
    method: 'post',
    data
  });
}

export function applySimpleWafConfig(data: SimpleWafConfigPayload) {
  return request<SimpleWafConfigResp>({
    url: '/api/caddy/waf/simple-config/apply',
    method: 'post',
    data
  });
}
