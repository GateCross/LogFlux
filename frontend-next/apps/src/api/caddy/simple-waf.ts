import { requestClient } from '#/api/request';

export namespace CaddySimpleWafApi {
  export interface SimpleWafConfig {
    [key: string]: any;
  }
}

/**
 * 获取简易 WAF 配置 — GET /caddy/waf/simple-config
 */
export async function getSimpleWafConfigApi() {
  return requestClient.get<CaddySimpleWafApi.SimpleWafConfig>(
    '/caddy/waf/simple-config',
  );
}

/**
 * 更新简易 WAF 配置 — PUT /caddy/waf/simple-config
 */
export async function updateSimpleWafConfigApi(
  data: CaddySimpleWafApi.SimpleWafConfig,
) {
  return requestClient.put<void>('/caddy/waf/simple-config', data);
}

/**
 * 预览简易 WAF 配置 — POST /caddy/waf/simple-config/preview
 */
export async function previewSimpleWafConfigApi(
  data: CaddySimpleWafApi.SimpleWafConfig,
) {
  return requestClient.post<CaddySimpleWafApi.SimpleWafConfig>(
    '/caddy/waf/simple-config/preview',
    data,
  );
}

/**
 * 应用简易 WAF 配置 — POST /caddy/waf/simple-config/apply
 */
export async function applySimpleWafConfigApi() {
  return requestClient.post<void>('/caddy/waf/simple-config/apply');
}
