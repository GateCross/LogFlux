import { requestClient } from '#/api/request';

export namespace CaddyServerApi {
  export interface CaddyServer {
    [key: string]: any;
  }

  export interface CaddyConfig {
    [key: string]: any;
  }

  export interface ConfigPreview {
    [key: string]: any;
  }

  export interface ConfigHistoryItem {
    [key: string]: any;
  }

  export interface RollbackParams {
    historyId: number;
  }

  export interface CaddyLogItem {
    [key: string]: any;
  }

  export interface CaddyLogPageResult {
    items: CaddyLogItem[];
    total: number;
  }
}

/**
 * 获取 Caddy 服务器列表 — GET /caddy/server
 */
export async function getCaddyServerListApi() {
  return requestClient.get<CaddyServerApi.CaddyServer[]>('/caddy/server');
}

/**
 * 添加 Caddy 服务器 — POST /caddy/server
 */
export async function addCaddyServerApi(data: CaddyServerApi.CaddyServer) {
  return requestClient.post<void>('/caddy/server', data);
}

/**
 * 更新 Caddy 服务器 — PUT /caddy/server/:id
 */
export async function updateCaddyServerApi(
  id: number,
  data: CaddyServerApi.CaddyServer,
) {
  return requestClient.put<void>(`/caddy/server/${id}`, data);
}

/**
 * 删除 Caddy 服务器 — DELETE /caddy/server/:id
 */
export async function deleteCaddyServerApi(id: number) {
  return requestClient.delete<void>(`/caddy/server/${id}`);
}

/**
 * 获取 Caddy 配置 — GET /caddy/server/:serverId/config
 */
export async function getCaddyConfigApi(serverId: number) {
  return requestClient.get<CaddyServerApi.CaddyConfig>(
    `/caddy/server/${serverId}/config`,
  );
}

/**
 * 推送 Caddy 原始配置 — POST /caddy/server/:serverId/config
 */
export async function pushCaddyConfigApi(
  serverId: number,
  data: CaddyServerApi.CaddyConfig,
) {
  return requestClient.post<void>(
    `/caddy/server/${serverId}/config`,
    data,
  );
}

/**
 * 预览 Caddy 配置变更 — POST /caddy/server/:serverId/config/preview
 */
export async function previewCaddyConfigApi(
  serverId: number,
  data: CaddyServerApi.CaddyConfig,
) {
  return requestClient.post<CaddyServerApi.ConfigPreview>(
    `/caddy/server/${serverId}/config/preview`,
    data,
  );
}

/**
 * 获取配置历史列表 — GET /caddy/server/:serverId/config/history
 */
export async function getCaddyConfigHistoryListApi(serverId: number) {
  return requestClient.get<CaddyServerApi.ConfigHistoryItem[]>(
    `/caddy/server/${serverId}/config/history`,
  );
}

/**
 * 获取配置历史详情 — GET /caddy/server/:serverId/config/history/:historyId
 */
export async function getCaddyConfigHistoryDetailApi(
  serverId: number,
  historyId: number,
) {
  return requestClient.get<CaddyServerApi.ConfigHistoryItem>(
    `/caddy/server/${serverId}/config/history/${historyId}`,
  );
}

/**
 * 回滚 Caddy 配置 — POST /caddy/server/:serverId/config/rollback
 */
export async function rollbackCaddyConfigApi(
  serverId: number,
  data: CaddyServerApi.RollbackParams,
) {
  return requestClient.post<void>(
    `/caddy/server/${serverId}/config/rollback`,
    data,
  );
}

/**
 * 查询 Caddy 访问日志（分页） — GET /caddy/logs
 */
export async function getCaddyLogsApi(params?: Record<string, any>) {
  return requestClient.get<CaddyServerApi.CaddyLogPageResult>('/caddy/logs', {
    params,
  });
}
