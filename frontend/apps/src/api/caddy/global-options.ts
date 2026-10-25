import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

export namespace CaddyGlobalOptionsApi {
  /**
   * 全局选项配置 — 对齐后端 GlobalOptions 契约。
   * autoHttps：'' 默认 / 'disable_redirects' 关闭跳转 / 'off' 完全关闭。
   */
  export interface GlobalOptions {
    autoHttps: string;
    debug: boolean;
    email: string;
    extraRawOptions: string;
    trustedProxies: string[];
  }

  /** 系统入口(:80) 配置，raw 为 Caddyfile 片段原文 */
  export interface EntryBlock {
    raw: string;
  }
}

/**
 * 获取全局选项 — GET /caddy/global-options
 */
export async function getGlobalOptionsApi(config?: RequestClientConfig) {
  return requestClient.get<CaddyGlobalOptionsApi.GlobalOptions>(
    '/caddy/global-options',
    config,
  );
}

/**
 * 更新全局选项 — PUT /caddy/global-options
 */
export async function updateGlobalOptionsApi(
  data: CaddyGlobalOptionsApi.GlobalOptions,
  config?: RequestClientConfig,
) {
  return requestClient.put<void>('/caddy/global-options', data, config);
}

/**
 * 获取系统入口(:80) 配置 — GET /caddy/entry-block
 */
export async function getEntryBlockApi(config?: RequestClientConfig) {
  return requestClient.get<CaddyGlobalOptionsApi.EntryBlock>(
    '/caddy/entry-block',
    config,
  );
}

/**
 * 更新系统入口(:80) 配置 — PUT /caddy/entry-block
 */
export async function updateEntryBlockApi(
  data: CaddyGlobalOptionsApi.EntryBlock,
  config?: RequestClientConfig,
) {
  return requestClient.put<void>('/caddy/entry-block', data, config);
}

/**
 * 恢复系统入口(:80) 默认模板 — POST /caddy/entry-block/reset-default
 */
export async function resetEntryBlockApi(config?: RequestClientConfig) {
  // 后端返回重置后的默认模板原文（CaddyEntryBlockResp）
  return requestClient.post<CaddyGlobalOptionsApi.EntryBlock>(
    '/caddy/entry-block/reset-default',
    undefined,
    config,
  );
}
