import { computed, ref } from 'vue';
import { useDialog, useMessage } from 'naive-ui';
import {
  addCaddyServer,
  deleteCaddyServer,
  fetchCaddyServers,
  updateCaddyServer
} from '@/service/api/caddy';

export interface CaddyServer {
  id: number;
  name: string;
  url: string;
  type: string;
  token?: string;
}

export function useCaddyServers(opts?: { onAllServersRemoved?: () => void }) {
  const message = useMessage();
  const dialog = useDialog();
  const servers = ref<CaddyServer[]>([]);
  const currentServerId = ref<number | null>(null);

  const serverOptions = computed(() => servers.value.map(s => ({ label: s.name, value: s.id })));

  // 服务器弹窗状态
  const showServerModal = ref(false);
  const serverModalType = ref<'add' | 'edit'>('add');
  const serverFormModel = ref<Omit<CaddyServer, 'id'> & { id?: number }>({
    name: '',
    url: '',
    type: 'local',
    token: ''
  });

  async function getServers() {
    const { data, error } = await fetchCaddyServers();
    if (error) {
      message.error('获取服务器列表失败');
      return;
    }
    if (data?.list) {
      servers.value = data.list;
      if (servers.value.length > 0) {
        if (!currentServerId.value || !servers.value.find(s => s.id === currentServerId.value)) {
          currentServerId.value = servers.value[0].id;
        }
      } else {
        currentServerId.value = null;
        opts?.onAllServersRemoved?.();
      }
    }
  }

  function openAddServerModal() {
    serverModalType.value = 'add';
    serverFormModel.value = { name: '', url: 'http://localhost:2019', type: 'local', token: '' };
    showServerModal.value = true;
  }

  function openEditServerModal() {
    const server = servers.value.find(s => s.id === currentServerId.value);
    if (!server) return;
    serverModalType.value = 'edit';
    serverFormModel.value = { ...server };
    showServerModal.value = true;
  }

  async function handleSaveServer() {
    let error;
    if (serverModalType.value === 'add') {
      const res = await addCaddyServer(serverFormModel.value);
      error = res.error;
    } else {
      const res = await updateCaddyServer(serverFormModel.value as CaddyServer);
      error = res.error;
    }
    if (error) {
      message.error('保存服务器失败');
      return;
    }
    message.success(serverModalType.value === 'add' ? '添加成功' : '更新成功');
    showServerModal.value = false;
    await getServers();
  }

  function handleDeleteServer() {
    if (!currentServerId.value) return;
    dialog.warning({
      title: '确认删除',
      content: '确定要删除此服务器吗？',
      positiveText: '确认',
      negativeText: '取消',
      onPositiveClick: async () => {
        const { error } = await deleteCaddyServer(currentServerId.value!);
        if (error) {
          message.error('删除服务器失败');
          return;
        }
        message.success('服务器已删除');
        await getServers();
      }
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
    handleDeleteServer
  };
}
