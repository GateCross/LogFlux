<script lang="ts" setup>
import {
  Alert,
  Button,
  Descriptions,
  DescriptionsItem,
  Drawer,
  Modal,
  Space,
  Spin,
  Switch,
  Tag,
} from 'antdv-next';

import type { CaddyServerApi } from '#/api/caddy/server';
import MonacoCodeEditor from '#/components/code-editor/MonacoCodeEditor.vue';
import { formatCaddyfile } from '../caddy-config-utils';

type DiffRow = {
  key: string | number;
  left: string | null;
  right: string | null;
  leftNo?: number | null;
  rightNo?: number | null;
  type: 'added' | 'changed' | 'removed' | 'same';
};

const props = defineProps<{
  open: boolean;
  historyLoading: boolean;
  historyList: CaddyServerApi.ConfigHistoryItem[];
  historyDescription: (item: CaddyServerApi.ConfigHistoryItem) => string;
  /** 读失败页内错误 */
  historyErrorMessage?: string | null;
  historyDetailVisible: boolean;
  historyCompareVisible: boolean;
  historyDetail: CaddyServerApi.ConfigHistoryDetail | null | undefined;
  historyDiffOnly: boolean;
  historyCompareRows: DiffRow[];
  diffSideClass: (
    row: { left: string | null; right: string | null; type: DiffRow['type'] },
    side: 'left' | 'right',
  ) => string;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'update:historyDetailVisible': [value: boolean];
  'update:historyCompareVisible': [value: boolean];
  'update:historyDiffOnly': [value: boolean];
  openDetail: [id: number];
  openCompare: [id: number];
  rollback: [id: number];
}>();
</script>

<template>
  <Drawer
    :open="props.open"
    title="配置历史"
    :size="720"
    @update:open="emit('update:open', $event)"
  >
    <Alert
      v-if="props.historyErrorMessage"
      class="mb-3"
      type="error"
      show-icon
      :message="props.historyErrorMessage"
    />
    <Spin :spinning="props.historyLoading">
      <div class="history-list">
        <div
          v-for="item in props.historyList"
          :key="item.id"
          class="history-item"
        >
          <div class="history-item-main">
            <div class="history-item-title">
              <Space>
                <span>{{ item.createdAt || `Version #${item.id}` }}</span>
                <Tag :color="item.action === 'rollback' ? 'orange' : 'blue'">
                  {{ item.action === 'rollback' ? '回滚' : '更新' }}
                </Tag>
              </Space>
            </div>
            <div class="history-item-desc">
              {{ props.historyDescription(item) }}
            </div>
          </div>
          <Space class="history-item-actions" :size="6">
            <Button
              size="small"
              class="table-action-btn"
              @click="emit('openDetail', item.id)"
            >
              查看
            </Button>
            <Button
              size="small"
              class="table-action-btn table-action-btn--secondary"
              @click="emit('openCompare', item.id)"
            >
              对比
            </Button>
            <Button
              size="small"
              danger
              class="table-action-btn table-action-btn--danger"
              @click="emit('rollback', item.id)"
            >
              回滚
            </Button>
          </Space>
        </div>
        <div v-if="props.historyList.length === 0" class="history-empty">
          暂无配置历史
        </div>
      </div>
    </Spin>
  </Drawer>

  <Modal
    :open="props.historyDetailVisible"
    width="900px"
    title="历史配置"
    :footer="null"
    @update:open="emit('update:historyDetailVisible', $event)"
  >
    <Descriptions v-if="props.historyDetail" bordered size="small" class="mb-3">
      <DescriptionsItem label="时间">{{ props.historyDetail.createdAt }}</DescriptionsItem>
      <DescriptionsItem label="动作">{{ props.historyDetail.action }}</DescriptionsItem>
      <DescriptionsItem label="Hash">{{ props.historyDetail.hash }}</DescriptionsItem>
    </Descriptions>
    <MonacoCodeEditor
      :model-value="formatCaddyfile(props.historyDetail?.config || '')"
      readonly
      language="plaintext"
      :height="480"
    />
  </Modal>

  <Modal
    :open="props.historyCompareVisible"
    width="1100px"
    title="历史对比"
    :footer="null"
    @update:open="emit('update:historyCompareVisible', $event)"
  >
    <div class="diff-toolbar">
      <span>左侧历史版本，右侧当前预览</span>
      <Switch
        :checked="props.historyDiffOnly"
        checked-children="仅差异"
        un-checked-children="全部"
        @update:checked="(checked) => emit('update:historyDiffOnly', Boolean(checked))"
      />
    </div>
    <div class="diff-grid">
      <div class="diff-column">
        <div
          v-for="row in props.historyCompareRows"
          :key="`l-${row.key}`"
          class="diff-line-row"
          :class="props.diffSideClass(row, 'left')"
        >
          <span class="diff-no">{{ row.leftNo ?? '' }}</span>
          <span class="diff-line">{{ row.left ?? '' }}</span>
        </div>
      </div>
      <div class="diff-column">
        <div
          v-for="row in props.historyCompareRows"
          :key="`r-${row.key}`"
          class="diff-line-row"
          :class="props.diffSideClass(row, 'right')"
        >
          <span class="diff-no">{{ row.rightNo ?? '' }}</span>
          <span class="diff-line">{{ row.right ?? '' }}</span>
        </div>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.mb-3 {
  margin-bottom: 12px;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.history-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.history-item:last-child {
  border-bottom: none;
}

.history-item-main {
  min-width: 0;
  flex: 1;
}

.history-item-title {
  margin-bottom: 4px;
  font-weight: 500;
}

.history-item-desc {
  color: #667085;
  font-size: 13px;
}

.history-item-actions {
  flex-shrink: 0;
}

.history-empty {
  padding: 24px 0;
  color: #98a2b3;
  text-align: center;
}

.diff-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.diff-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  overflow: hidden;
}

.diff-column {
  max-height: 65vh;
  overflow: auto;
}

.diff-column + .diff-column {
  border-left: 1px solid #e5e7eb;
}

.diff-line-row {
  display: grid;
  grid-template-columns: 40px 1fr;
  gap: 8px;
  padding: 5px 8px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  white-space: pre-wrap;
  border-left: 3px solid transparent;
}

.diff-line-row.added {
  color: #166534;
  background: #dcfce7;
  border-left-color: #16a34a;
}

.diff-line-row.removed {
  color: #991b1b;
  background: #fee2e2;
  border-left-color: #b91c1c;
}

.diff-line-row.blank {
  color: transparent;
  background: #f8fafc;
}

.diff-no {
  color: #98a2b3;
  text-align: right;
  user-select: none;
}

.diff-line {
  overflow-wrap: anywhere;
}
</style>
