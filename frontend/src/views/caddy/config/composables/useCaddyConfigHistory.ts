import { computed, h, ref, type Ref } from 'vue';
import { NButton, NTag, useDialog, useMessage } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import {
  fetchCaddyConfigHistory,
  fetchCaddyConfigHistoryDetail,
  rollbackCaddyConfig
} from '@/service/api/caddy';
import { formatCaddyfile } from '../caddy-config-utils';
import type { CaddyConfigHistoryItem } from '@/service/api/caddy';

export function useCaddyConfigHistory(opts: {
  currentServerId: Ref<number | null>;
  formattedConfigContent: Ref<string>;
  loadConfig: () => Promise<void>;
}) {
  const message = useMessage();
  const dialog = useDialog();

  const showHistoryModal = ref(false);
  const historyLoading = ref(false);
  const historyList = ref<CaddyConfigHistoryItem[]>([]);
  const historyPagination = ref({ page: 1, pageSize: 10, itemCount: 0 });

  const showHistoryDetailModal = ref(false);
  const showHistoryCompareModal = ref(false);
  const historyDetail = ref<{
    id: number;
    createdAt: string;
    action: string;
    hash: string;
    config: string;
  } | null>(null);
  const historyCompareLeft = ref('');
  const historyDiffOnly = ref(false);

  const historyDetailFormattedConfig = computed(() =>
    historyDetail.value ? formatCaddyfile(historyDetail.value.config) : ''
  );
  const historyCompareLeftFormatted = computed(() => formatCaddyfile(historyCompareLeft.value));

  function formatHistoryAction(action: string) {
    return action === 'rollback' ? '回滚' : '更新';
  }

  const historyColumns: DataTableColumns<CaddyConfigHistoryItem> = [
    { title: '时间', key: 'createdAt', width: 180 },
    {
      title: '动作',
      key: 'action',
      width: 100,
      render(row) {
        const label = formatHistoryAction(row.action);
        const type = row.action === 'rollback' ? 'warning' : 'info';
        return h(NTag, { type, size: 'small' }, { default: () => label });
      }
    },
    {
      title: '操作',
      key: 'actions',
      width: 220,
      render(row) {
        return h('div', { class: 'flex gap-2' }, [
          h(NButton, { size: 'tiny', onClick: () => openHistoryDetail(row.id) }, { default: () => '查看' }),
          h(NButton, { size: 'tiny', onClick: () => openHistoryCompare(row.id) }, { default: () => '对比' }),
          h(NButton, { size: 'tiny', type: 'primary', onClick: () => handleRollback(row.id) }, { default: () => '回滚' })
        ]);
      }
    }
  ];

  async function openHistoryModal() {
    if (!opts.currentServerId.value) return;
    showHistoryModal.value = true;
    historyPagination.value.page = 1;
    await fetchHistory();
  }

  async function fetchHistory() {
    if (!opts.currentServerId.value) return;
    historyLoading.value = true;
    const { data, error } = await fetchCaddyConfigHistory(opts.currentServerId.value, {
      page: historyPagination.value.page,
      pageSize: historyPagination.value.pageSize
    });
    historyLoading.value = false;
    if (error) {
      message.error('获取历史版本失败');
      return;
    }
    historyList.value = data?.list || [];
    historyPagination.value.itemCount = data?.total || 0;
  }

  async function fetchHistoryDetail(historyId: number) {
    if (!opts.currentServerId.value) return null;
    const { data, error } = await fetchCaddyConfigHistoryDetail(opts.currentServerId.value, historyId);
    if (error) {
      message.error('获取历史配置失败');
      return null;
    }
    return data;
  }

  async function openHistoryDetail(historyId: number) {
    const detail = await fetchHistoryDetail(historyId);
    if (!detail) return;
    historyDetail.value = {
      id: detail.id,
      createdAt: detail.createdAt,
      action: detail.action,
      hash: detail.hash,
      config: detail.config || ''
    };
    showHistoryDetailModal.value = true;
  }

  async function openHistoryCompare(historyId: number) {
    const detail = await fetchHistoryDetail(historyId);
    if (!detail) return;
    historyCompareLeft.value = detail.config || '';
    historyDiffOnly.value = false;
    showHistoryCompareModal.value = true;
  }

  function handleHistoryPageChange(page: number) {
    historyPagination.value.page = page;
    fetchHistory();
  }

  async function handleRollback(historyId: number) {
    if (!opts.currentServerId.value) return;
    dialog.warning({
      title: '确认回滚',
      content: '确定要回滚到该版本吗？',
      positiveText: '确认',
      negativeText: '取消',
      onPositiveClick: async () => {
        const { error } = await rollbackCaddyConfig(opts.currentServerId.value!, historyId);
        if (error) {
          message.error('回滚失败');
          return;
        }
        message.success('回滚成功');
        await opts.loadConfig();
        await fetchHistory();
      }
    });
  }

  return {
    showHistoryModal,
    historyLoading,
    historyList,
    historyPagination,
    showHistoryDetailModal,
    showHistoryCompareModal,
    historyDetail,
    historyCompareLeft,
    historyDiffOnly,
    historyDetailFormattedConfig,
    historyCompareLeftFormatted,
    historyColumns,
    formatHistoryAction,
    openHistoryModal,
    handleHistoryPageChange
  };
}
