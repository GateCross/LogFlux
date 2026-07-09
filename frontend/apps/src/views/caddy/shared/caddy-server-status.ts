/**
 * Caddy 节点状态探测 — 纯函数工具。
 * 配置管理与服务目录共用展示语义；探测仍走 GET /caddy/server/status。
 */

import type { CaddyServerApi } from '#/api/caddy/server';

export type ServerStatusItem = CaddyServerApi.CaddyServerStatusItem;

export function formatLatency(ms: number | undefined | null): string {
  if (ms === undefined || ms === null || Number.isNaN(Number(ms))) return '—';
  return `${Math.max(0, Math.round(Number(ms)))} ms`;
}

export function statusTagColor(status?: ServerStatusItem | null): string {
  if (!status) return 'default';
  return status.reachable ? 'success' : 'error';
}

export function statusLabel(status?: ServerStatusItem | null): string {
  if (!status) return '未探测';
  return status.reachable ? '在线' : '离线';
}

export function statusErrorSummary(status?: ServerStatusItem | null): string {
  if (!status) return '';
  if (status.reachable) return '';
  return (status.errorMessage || '探测失败').trim();
}

export function statusTooltip(status?: ServerStatusItem | null): string {
  if (!status) return '尚未探测节点状态';
  const err = statusErrorSummary(status);
  if (err) return err;
  return `延迟 ${formatLatency(status.latencyMs)} · ${status.probedAt || ''}`;
}

/** 将节点列表与探测结果合并为列表行（无探测时 status 为空） */
export function mergeServerStatusRows(
  servers: Array<Record<string, any>>,
  statusList: ServerStatusItem[],
  labelOf: (server: Record<string, any>) => string,
): Array<{
  id: number;
  name: string;
  url: string;
  status?: ServerStatusItem;
}> {
  const map = new Map<number, ServerStatusItem>();
  for (const item of statusList) {
    map.set(Number(item.serverId), item);
  }
  return servers.map((server) => {
    const id = Number(server.id);
    return {
      id,
      name: labelOf(server),
      url: String(server.url ?? ''),
      status: map.get(id),
    };
  });
}

export function pickLatestProbedAt(list: ServerStatusItem[]): string {
  const latest = list
    .map((item) => item.probedAt)
    .filter(Boolean)
    .sort()
    .at(-1);
  return latest || new Date().toLocaleString('zh-CN', { hour12: false });
}
