import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

export namespace CaddyIpRegionApi {
  /**
   * IP 地域访问控制配置 — 对齐 backend IPRegionConfigResp / UpdateReq。
   * 读写字段一致，因此复用同一显式契约。
   */
  export interface IpRegionConfig {
    allowList: string[];
    enabled: boolean;
  }
}

/**
 * 获取 IP 地域配置 — GET /caddy/ip-region
 */
export async function getIpRegionConfigApi(config?: RequestClientConfig) {
  return requestClient.get<CaddyIpRegionApi.IpRegionConfig>(
    '/caddy/ip-region',
    config,
  );
}

/**
 * 更新 IP 地域配置 — PUT /caddy/ip-region
 */
export async function updateIpRegionConfigApi(
  data: CaddyIpRegionApi.IpRegionConfig,
) {
  return requestClient.put<void>('/caddy/ip-region', data);
}
