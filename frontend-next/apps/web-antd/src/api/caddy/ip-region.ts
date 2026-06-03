import { requestClient } from '#/api/request';

export namespace CaddyIpRegionApi {
  export interface IpRegionConfig {
    [key: string]: any;
  }
}

/**
 * 获取 IP 地域配置 — GET /caddy/ip-region
 */
export async function getIpRegionConfigApi() {
  return requestClient.get<CaddyIpRegionApi.IpRegionConfig>(
    '/caddy/ip-region',
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
