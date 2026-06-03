import { computed, ref } from 'vue';
import { Modal, message } from 'ant-design-vue';
import {
  getCaddyServerListApi,
  addCaddyServerApi,
  updateCaddyServerApi,
  deleteCaddyServerApi,
} from '#/api/caddy/server';

export interface CaddyServer {
  id: number;
  name: string;
  url: string;
  type: string;
  token?: string;
}

export function useCaddyServers(opts?: { onAllServersRemoved?: () => void }) {
  const servers = ref<CaddyServer[]>([]);
  const currentServerId = ref<number | null>(null);

  const serverOptions = computed(() =>
    servers.value.map((s) => ({ label: s.name, value: s.id })),
  );

  const showServerModal = ref(false);
  const serverModalType = ref<'add' | 'edit'>('add');
  const serverFormModel = ref<Omit<CaddyServer, 'id'> & { id?: number }>({
    name: '',
    url: '',
    type: 'local',
    token: '',
  });

  async function getServers() {
    try {
      const list = await getCaddyServerListApi();
      servers.value = Array.isArray(list) ? list : [];
      if (servers.value.length > 0) {
        if (
          !currentServerId.value ||
          !servers.value.find((s) => s.id === currentServerId.value)
        ) {
          currentServerId.value = servers.value[0]!.id;
        }
      } else {
        currentServerId.value = null;
        opts?.onAllServersRemoved?.();
      }
    } catch {
      message.error('获取服务器列表失败');
    }
  }

  function openAddServerModal() {
    serverModalType.value = 'add';
    serverFormModel.value = {
      name: '',
      url: 'http://localhost:2019',
      type: 'local',
      token: '',
    };
    showServerModal.value = true;
  }

  function openEditServerModal() {
    const server = servers.value.find(
      (s) => s.id === currentServerId.value,
    );
    if (!server) return;
    serverModalType.value = 'edit';
    serverFormModel.value = { ...server };
    showServerModal.value = true;
  }

  async function handleSaveServer() {
    try {
      if (serverModalType.value === 'add') {
        await addCaddyServerApi(serverFormModel.value as any);
      } else {
        await updateCaddyServerApi(
          serverFormModel.value.id!,
          serverFormModel.value as any,
        );
      }
      message.success(
        serverModalType.value === 'add' ? '添加成功' : '更新成功',
      );
      showServerModal.value = false;
      await getServers();
    } catch {
      message.error('保存服务器失败');
    }
  }

  function handleDeleteServer() {
    if (!currentServerId.value) return;
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除此服务器吗？',
      okText: '确认',
      cancelText: '取消',
      async onOk() {
        try {
          await deleteCaddyServerApi(currentServerId.value!);
          message.success('服务器已删除');
          await getServers();
        } catch {
          message.error('删除服务器失败');
        }
      },
    });
  }

  return {
    servers,
    currentServerId,
    serverOptions,
    showServerModal,
    serverModalType,
    serverFormModel,
    getServers,
    openAddServerModal,
    openEditServerModal,
    handleSaveServer,
    handleDeleteServer,
  };
}
