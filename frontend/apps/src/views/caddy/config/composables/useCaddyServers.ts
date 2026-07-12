import { computed, reactive, ref, watch } from 'vue';
import { message, Modal } from 'antdv-next';
import { useQueryClient } from '@tanstack/vue-query';

import {
  addCaddyServerApi,
  deleteCaddyServerApi,
  getCaddyServerListApi,
  updateCaddyServerApi,
} from '#/api/caddy/server';
import type { CaddyServerApi } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

import { useCaddyServerStatus } from '../../shared/use-caddy-server-status';

/** 服务器新增/编辑表单 — 对齐 Create/Update，禁止开放字典 */
export type ServerFormState = {
  id?: number;
  name: string;
  token: string;
  type: string;
  url: string;
};

export function createEmptyServerForm(): ServerFormState {
  return {
    id: undefined,
    name: '',
    token: '',
    type: 'local',
    url: 'http://localhost:2019',
  };
}

export function serverLabel(server: Record<string, any>) {
  return server.name || server.url || `Server #${server.id}`;
}

export interface UseCaddyServersOptions {
  /** 解析深链 serverId（如 /caddy/config?serverId=） */
  resolvePreferredServerId?: () => number | undefined;
}

export function useCaddyServers(options: UseCaddyServersOptions = {}) {
  const { resolvePreferredServerId } = options;
  const queryClient = useQueryClient();

  const selectedServerId = ref<number>();
  const serverModalVisible = ref(false);
  const serverModalType = ref<'add' | 'edit'>('add');
  const serverForm = reactive<ServerFormState>(createEmptyServerForm());

  const {
    data: serversData,
    loading: loadingServers,
    errorMessage: serversErrorMessage,
    refetch: refetchServers,
  } = useListDetailQuery({
    queryKey: qk.caddy.servers(),
    queryFn: () => getCaddyServerListApi(withListDetailErrorMode()),
    errorFallback: '获取服务器列表失败',
  });

  const servers = computed(() => serversData.value ?? []);

  /**
   * 配置管理：仅保留当前选中节点的轻量状态提示。
   * 完整节点状态总览在「服务目录」页。
   */
  const {
    loadingServerStatus,
    serverStatusLoaded,
    selectedServerStatus,
    pruneByServers,
    handleRefreshServerStatus,
  } = useCaddyServerStatus({
    servers,
    selectedServerId,
    labelOf: serverLabel,
  });

  watch(
    servers,
    (list) => {
      const preferredId = resolvePreferredServerId?.();
      if (
        preferredId &&
        list.some((server) => Number(server.id) === preferredId)
      ) {
        selectedServerId.value = preferredId;
      } else if (list.length > 0 && !selectedServerId.value) {
        selectedServerId.value = Number(list[0]?.id);
      }
      pruneByServers(list);
    },
    { immediate: true },
  );

  const serverOptions = computed(() =>
    servers.value.map((server) => ({
      label: serverLabel(server),
      value: Number(server.id),
    })),
  );

  const selectedServer = computed(() =>
    servers.value.find((server) => Number(server.id) === selectedServerId.value),
  );

  async function fetchServers() {
    await refetchServers();
  }

  function handleServerChange(value: unknown) {
    const next = Number(value);
    if (!Number.isFinite(next)) return;
    selectedServerId.value = next;
  }

  function openAddServerModal() {
    serverModalType.value = 'add';
    Object.assign(serverForm, createEmptyServerForm());
    serverModalVisible.value = true;
  }

  function openEditServerModal() {
    if (!selectedServer.value) return;
    serverModalType.value = 'edit';
    Object.assign(serverForm, {
      id: selectedServer.value.id,
      name: selectedServer.value.name ?? '',
      // token 不在列表模型中；编辑时留空表示不修改
      token: '',
      type: selectedServer.value.type ?? 'local',
      url: selectedServer.value.url ?? '',
    } satisfies ServerFormState);
    serverModalVisible.value = true;
  }

  async function saveServer() {
    if (!serverForm.name || !serverForm.url) {
      message.warning('请填写服务器名称和地址');
      return;
    }
    try {
      if (serverModalType.value === 'add') {
        const payload: CaddyServerApi.CaddyServerCreate = {
          name: serverForm.name,
          url: serverForm.url,
          type: serverForm.type,
          token: serverForm.token || undefined,
        };
        await addCaddyServerApi(payload);
        message.success('服务器已添加');
      } else if (serverForm.id) {
        const payload: CaddyServerApi.CaddyServerUpdate = {
          name: serverForm.name,
          url: serverForm.url,
          type: serverForm.type,
          token: serverForm.token || undefined,
        };
        await updateCaddyServerApi(Number(serverForm.id), payload);
        message.success('服务器已更新');
      }
      serverModalVisible.value = false;
      await invalidateListDetailQueries(queryClient, qk.caddy.servers());
    } catch {
      message.error('保存服务器失败');
    }
  }

  function deleteCurrentServer() {
    if (!selectedServerId.value) return;
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除当前 Caddy 服务器吗？',
      async onOk() {
        try {
          await deleteCaddyServerApi(selectedServerId.value!);
          message.success('服务器已删除');
          selectedServerId.value = undefined;
          await invalidateListDetailQueries(queryClient, qk.caddy.servers());
        } catch {
          message.error('删除服务器失败');
        }
      },
    });
  }

  return {
    servers,
    selectedServerId,
    loadingServers,
    serversErrorMessage,
    serverModalVisible,
    serverModalType,
    serverForm,
    serverOptions,
    selectedServer,
    loadingServerStatus,
    serverStatusLoaded,
    selectedServerStatus,
    handleRefreshServerStatus,
    fetchServers,
    handleServerChange,
    openAddServerModal,
    openEditServerModal,
    saveServer,
    deleteCurrentServer,
    serverLabel,
  };
}
