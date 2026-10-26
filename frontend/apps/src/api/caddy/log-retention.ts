import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

export namespace CaddyLogRetentionApi {
  /** 站点日志保留策略 — 对齐后端 CaddyLogRetentionResp */
  export interface LogRetention {
    /** 是否启用自动清理；关闭后保留天数仅作为展示值，不执行清理 */
    enabled: boolean;
    /** 保留天数；启用后早于该天数的 caddy_logs 记录每天清理一次 */
    retentionDays: number;
  }
}

/**
 * 获取站点日志保留天数 — GET /caddy/log-retention
 */
export async function getLogRetentionApi(config?: RequestClientConfig) {
  return requestClient.get<CaddyLogRetentionApi.LogRetention>(
    '/caddy/log-retention',
    config,
  );
}

/**
 * 更新站点日志保留天数 — PUT /caddy/log-retention
 */
export async function updateLogRetentionApi(
  data: CaddyLogRetentionApi.LogRetention,
  config?: RequestClientConfig,
) {
  return requestClient.put<void>('/caddy/log-retention', data, config);
}
