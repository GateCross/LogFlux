import { computed, ref, type Ref } from 'vue';
import { message, Modal } from 'antdv-next';
import { useQueryClient } from '@tanstack/vue-query';

import {
  getCaddyConfigHistoryDetailApi,
  getCaddyConfigHistoryListApi,
  rollbackCaddyConfigApi,
} from '#/api/caddy/server';
import type { CaddyServerApi } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueryKeys } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { apiErrorMessage } from '#/utils/api-error-message';

import { buildLineDiff, formatCaddyfile } from '../caddy-config-utils';

export function shortHash(value: unknown) {
  const text = String(value ?? '').trim();
  if (text.length <= 16) return text;
  return `${text.slice(0, 8)}...${text.slice(-8)}`;
}

export function historyDescription(item: CaddyServerApi.ConfigHistoryItem) {
  return shortHash(item.hash);
}

export function diffSideClass(
  row: { left: string | null; right: string | null; type: 'added' | 'changed' | 'removed' | 'same' },
  side: 'left' | 'right',
) {
  if (row.type === 'same') return 'same';
  if (side === 'left') {
    if (row.left === null) return 'blank';
    return 'removed';
  }
  if (row.right === null) return 'blank';
  return 'added';
}

export interface UseConfigHistoryOptions {
  selectedServerId: Ref<number | undefined>;
  /** 当前将要发布的配置文本（对比右侧） */
  previewConfig: Ref<string> | { value: string };
  /** 回滚成功后刷新配置 */
  onRolledBack?: () => Promise<void> | void;
}

export function useConfigHistory(options: UseConfigHistoryOptions) {
  const { selectedServerId, previewConfig, onRolledBack } = options;
  const queryClient = useQueryClient();

  const historyDrawerVisible = ref(false);
  const historyLoading = ref(false);
  const historyList = ref<CaddyServerApi.ConfigHistoryItem[]>([]);
  const historyDetailVisible = ref(false);
  const historyCompareVisible = ref(false);
  const historyDetail = ref<CaddyServerApi.ConfigHistoryDetail | null>(null);
  const historyCompareLeft = ref('');
  const historyDiffOnly = ref(false);
  /** 历史读失败页内展示（suppress 全局 toast 后不再 message.error） */
  const historyErrorMessage = ref<string | null>(null);

  const historyCompareRows = computed(() => {
    const rows = buildLineDiff(
      formatCaddyfile(historyCompareLeft.value || ''),
      // previewConfig 已在 configIO 中 format 过，保持与拆分前一致
      previewConfig.value || '',
    );
    return historyDiffOnly.value ? rows.filter((row) => row.type !== 'same') : rows;
  });

  /** 回滚改写服务端配置：失效 config（含 simple-waf）、history、catalog */
  async function invalidateAfterConfigMutation(serverId: number) {
    await invalidateListDetailQueryKeys(queryClient, [
      qk.caddy.config(serverId),
      ['caddy', 'history', serverId],
      qk.caddy.catalog(serverId),
      ['caddy', 'metrics', serverId],
    ]);
  }

  async function openHistory() {
    if (!selectedServerId.value) return;
    historyDrawerVisible.value = true;
    historyLoading.value = true;
    historyErrorMessage.value = null;
    try {
      const data = await queryClient.fetchQuery({
        queryKey: qk.caddy.history(selectedServerId.value, 1, 50),
        queryFn: () =>
          getCaddyConfigHistoryListApi(
            selectedServerId.value!,
            withListDetailErrorMode(),
          ),
      });
      historyList.value = data?.list ?? [];
    } catch (error) {
      historyErrorMessage.value = apiErrorMessage(error, '获取历史版本失败');
    } finally {
      historyLoading.value = false;
    }
  }

  async function openHistoryDetail(id: number) {
    if (!selectedServerId.value) return;
    historyErrorMessage.value = null;
    try {
      historyDetail.value = await queryClient.fetchQuery({
        queryKey: [
          ...qk.caddy.history(selectedServerId.value, 1, 1),
          'detail',
          id,
        ],
        queryFn: () =>
          getCaddyConfigHistoryDetailApi(
            selectedServerId.value!,
            id,
            withListDetailErrorMode(),
          ),
      });
      historyDetailVisible.value = true;
    } catch (error) {
      historyErrorMessage.value = apiErrorMessage(error, '获取历史配置失败');
    }
  }

  async function openHistoryCompare(id: number) {
    if (!selectedServerId.value) return;
    historyErrorMessage.value = null;
    try {
      const detail = await queryClient.fetchQuery({
        queryKey: [
          ...qk.caddy.history(selectedServerId.value, 1, 1),
          'detail',
          id,
        ],
        queryFn: () =>
          getCaddyConfigHistoryDetailApi(
            selectedServerId.value!,
            id,
            withListDetailErrorMode(),
          ),
      });
      historyCompareLeft.value = detail?.config ?? '';
      historyDiffOnly.value = false;
      historyCompareVisible.value = true;
    } catch (error) {
      historyErrorMessage.value = apiErrorMessage(error, '获取历史配置失败');
    }
  }

  function rollbackHistory(id: number) {
    if (!selectedServerId.value) return;
    const serverId = selectedServerId.value;
    Modal.confirm({
      title: '确认回滚',
      content: '确定要回滚到该版本吗？',
      async onOk() {
        try {
          await rollbackCaddyConfigApi(serverId, { historyId: id });
          message.success('回滚成功');
          // 先失效相关缓存，再 fetchConfig() / openHistory，避免 staleTime 命中旧数据
          await invalidateAfterConfigMutation(serverId);
          await onRolledBack?.();
          await openHistory();
        } catch (error) {
          // 写路径保留单次 toast
          message.error(apiErrorMessage(error, '回滚失败'));
        }
      },
    });
  }

  return {
    historyDrawerVisible,
    historyLoading,
    historyList,
    historyDetailVisible,
    historyCompareVisible,
    historyDetail,
    historyCompareLeft,
    historyDiffOnly,
    historyCompareRows,
    historyErrorMessage,
    shortHash,
    historyDescription,
    diffSideClass,
    openHistory,
    openHistoryDetail,
    openHistoryCompare,
    rollbackHistory,
  };
}
