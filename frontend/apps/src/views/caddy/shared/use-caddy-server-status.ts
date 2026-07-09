/**
 * Caddy 节点状态探测 composable。
 * - 仅手动刷新 + debounce，禁止导航 thrash 自动全量探测
 * - 配置管理：只读当前选中节点摘要
 * - 服务目录：完整节点状态总览
 */

import { computed, ref, type Ref } from 'vue';

import { useDebounceFn } from '@vueuse/core';
import { message } from 'ant-design-vue';

import { getCaddyServerStatusApi } from '#/api/caddy/server';

import {
  mergeServerStatusRows,
  pickLatestProbedAt,
  type ServerStatusItem,
} from './caddy-server-status';

export interface UseCaddyServerStatusOptions {
  /** 已登记节点列表（与探测结果合并） */
  servers: Ref<Array<Record<string, any>>>;
  /** 当前选中节点 id；配置管理用于轻量摘要 */
  selectedServerId?: Ref<number | undefined>;
  /** 节点展示名 */
  labelOf?: (server: Record<string, any>) => string;
  /** 探测失败是否弹 toast；默认 true */
  toastOnError?: boolean;
}

function defaultServerLabel(server: Record<string, any>) {
  return server.name ?? server.host ?? server.url ?? `Server #${server.id}`;
}

export function useCaddyServerStatus(options: UseCaddyServerStatusOptions) {
  const {
    servers,
    selectedServerId,
    labelOf = defaultServerLabel,
    toastOnError = true,
  } = options;

  const serverStatusList = ref<ServerStatusItem[]>([]);
  const loadingServerStatus = ref(false);
  const serverStatusError = ref('');
  const serverStatusLoaded = ref(false);
  const lastProbedAt = ref('');

  const serverStatusMap = computed(() => {
    const map = new Map<number, ServerStatusItem>();
    for (const item of serverStatusList.value) {
      map.set(Number(item.serverId), item);
    }
    return map;
  });

  const selectedServerStatus = computed(() => {
    if (!selectedServerId?.value) return undefined;
    return serverStatusMap.value.get(selectedServerId.value);
  });

  const serverStatusRows = computed(() =>
    mergeServerStatusRows(servers.value, serverStatusList.value, labelOf),
  );

  const onlineCount = computed(
    () => serverStatusList.value.filter((s) => s.reachable).length,
  );
  const offlineCount = computed(
    () => serverStatusList.value.filter((s) => !s.reachable).length,
  );

  /** 节点被删后清理无效探测缓存，避免幽灵状态 */
  function pruneByServers(nextServers: Array<Record<string, any>>) {
    if (nextServers.length === 0) {
      serverStatusList.value = [];
      serverStatusLoaded.value = false;
      lastProbedAt.value = '';
      serverStatusError.value = '';
      return;
    }
    if (serverStatusList.value.length === 0) return;
    const ids = new Set(nextServers.map((s) => Number(s.id)));
    serverStatusList.value = serverStatusList.value.filter((item) =>
      ids.has(Number(item.serverId)),
    );
  }

  /**
   * 只读探测：GET /caddy/server/status。
   * 仅由手动「探测状态」触发（带 debounce），禁止在路由/导航 thrash 时自动全量探测。
   */
  async function fetchServerStatus() {
    if (loadingServerStatus.value) return;
    loadingServerStatus.value = true;
    serverStatusError.value = '';
    try {
      const list = await getCaddyServerStatusApi();
      serverStatusList.value = Array.isArray(list) ? list : [];
      serverStatusLoaded.value = true;
      lastProbedAt.value = pickLatestProbedAt(serverStatusList.value);
      if (serverStatusList.value.length === 0 && servers.value.length === 0) {
        serverStatusError.value = '';
      }
    } catch (error) {
      const data = (error as any)?.response?.data ?? (error as any)?.data ?? {};
      const detail =
        data?.message ?? data?.msg ?? data?.error ?? (error as any)?.message;
      serverStatusError.value =
        typeof detail === 'string' && detail.trim()
          ? detail.trim()
          : '探测节点状态失败';
      if (toastOnError) {
        message.error(serverStatusError.value);
      }
    } finally {
      loadingServerStatus.value = false;
    }
  }

  const debouncedFetchServerStatus = useDebounceFn(fetchServerStatus, 400);

  function handleRefreshServerStatus() {
    debouncedFetchServerStatus();
  }

  return {
    serverStatusList,
    loadingServerStatus,
    serverStatusError,
    serverStatusLoaded,
    lastProbedAt,
    serverStatusMap,
    selectedServerStatus,
    serverStatusRows,
    onlineCount,
    offlineCount,
    pruneByServers,
    fetchServerStatus,
    handleRefreshServerStatus,
  };
}
