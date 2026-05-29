import { request } from '../request';

export interface IPRegionConfig {
  enabled: boolean;
  allowList: string[];
}

export function fetchIPRegionConfig() {
  return request<IPRegionConfig>({ url: '/api/caddy/ip-region' });
}

export function updateIPRegionConfig(data: IPRegionConfig) {
  return request<IPRegionConfig>({ url: '/api/caddy/ip-region', method: 'put', data });
}
