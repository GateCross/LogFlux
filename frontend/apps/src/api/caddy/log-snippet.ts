import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

export namespace CaddyLogSnippetApi {
  /**
   * 站点日志 snippet 定义 — 对齐后端 CaddyLogSnippetResp。
   * raw 是 (subdomain-log) 定义的 Caddyfile 原文，整段由 LogFlux 托管。
   */
  export interface LogSnippet {
    raw: string;
  }
}

/**
 * 获取日志 snippet 定义 — GET /caddy/log-snippet
 */
export async function getLogSnippetApi(config?: RequestClientConfig) {
  return requestClient.get<CaddyLogSnippetApi.LogSnippet>(
    '/caddy/log-snippet',
    config,
  );
}

/**
 * 更新日志 snippet 定义 — PUT /caddy/log-snippet
 * 后端只落库不触发 Apply，需在配置管理页整体应用后才合并进 Caddyfile。
 */
export async function updateLogSnippetApi(
  data: CaddyLogSnippetApi.LogSnippet,
  config?: RequestClientConfig,
) {
  return requestClient.put<CaddyLogSnippetApi.LogSnippet>(
    '/caddy/log-snippet',
    data,
    config,
  );
}

/**
 * 恢复默认日志 snippet 定义 — POST /caddy/log-snippet/reset-default
 * 后端返回重置后的默认模板原文（CaddyLogSnippetResp）。
 */
export async function resetLogSnippetApi(config?: RequestClientConfig) {
  return requestClient.post<CaddyLogSnippetApi.LogSnippet>(
    '/caddy/log-snippet/reset-default',
    undefined,
    config,
  );
}
