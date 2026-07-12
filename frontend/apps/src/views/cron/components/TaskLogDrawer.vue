<script lang="ts" setup>
import type { CronApi } from '#/api/cron';

import { computed } from 'vue';

import {
  Button,
  Descriptions,
  DescriptionsItem,
  Drawer,
  Modal,
  Table,
  Tag,
  Input,
} from 'antdv-next';

import {
  formatScriptMode,
  formatStatus,
  formatTriggerMode,
  statusColor,
} from '../utils';

const props = defineProps<{
  open: boolean;
  title: string;
  logs: CronApi.Log[];
  logsLoading: boolean;
  pagination: {
    current: number;
    pageSize: number;
    total: number;
  };
  detailVisible: boolean;
  detailLoading: boolean;
  currentLog: CronApi.Log | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'update:detailVisible': [value: boolean];
  refresh: [];
  tableChange: [pag: { current: number; pageSize: number }];
  openDetail: [id: number];
}>();

const logColumns = [
  { dataIndex: 'id', key: 'id', title: 'ID', width: 80 },
  { dataIndex: 'taskName', key: 'taskName', title: '任务名称', ellipsis: true },
  { dataIndex: 'startTime', key: 'startTime', title: '开始时间', width: 170 },
  { dataIndex: 'triggerMode', key: 'triggerMode', title: '触发方式', width: 110 },
  { dataIndex: 'scriptMode', key: 'scriptMode', title: '脚本来源', width: 110 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 90 },
  { dataIndex: 'duration', key: 'duration', title: '耗时', width: 100 },
  { key: 'actions', title: '操作', width: 90 },
];

const logDetailTitle = computed(() =>
  props.currentLog ? `日志详情 #${props.currentLog.id}` : '日志详情',
);

const outputLineCount = computed(() => {
  const output = props.currentLog?.output ?? '';
  return output.split(/\r?\n/).filter(Boolean).length;
});

function onTableChange(pag: { current?: number; pageSize?: number }) {
  emit('tableChange', {
    current: pag.current ?? 1,
    pageSize: pag.pageSize ?? 20,
  });
}
</script>

<template>
  <Drawer
    :open="open"
    :title="title"
    :size="960"
    :styles="{ body: { padding: '16px' } }"
    @update:open="emit('update:open', $event)"
  >
    <div class="mb-3 flex justify-end">
      <Button @click="emit('refresh')">刷新</Button>
    </div>
    <Table
      :columns="logColumns"
      :data-source="logs"
      :loading="logsLoading"
      :pagination="{
        current: pagination.current,
        pageSize: pagination.pageSize,
        total: pagination.total,
        showSizeChanger: true,
        showTotal: (total: number) => `共 ${total} 条`,
      }"
      row-key="id"
      size="small"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'triggerMode'">
          <Tag :color="record.triggerMode === 'schedule' ? 'purple' : 'orange'">
            {{ formatTriggerMode(record.triggerMode) }}
          </Tag>
        </template>
        <template v-if="column.key === 'scriptMode'">
          <Tag :color="record.scriptMode === 'file' ? 'green' : 'default'">
            {{ formatScriptMode(record.scriptMode) }}
          </Tag>
        </template>
        <template v-if="column.key === 'status'">
          <Tag :color="statusColor(record.status)">
            {{ formatStatus(record.status) }}
          </Tag>
        </template>
        <template v-if="column.key === 'duration'">
          {{ record.duration }} ms
        </template>
        <template v-if="column.key === 'actions'">
          <Button
            size="small"
            class="table-action-btn"
            @click="emit('openDetail', record.id)"
          >
            详情
          </Button>
        </template>
      </template>
    </Table>
  </Drawer>

  <Modal
    :open="detailVisible"
    :title="logDetailTitle"
    :width="960"
    footer=""
    @cancel="emit('update:detailVisible', false)"
  >
    <div v-if="detailLoading" class="py-8 text-center text-gray-500">加载中...</div>
    <div v-else-if="currentLog" class="space-y-4">
      <Descriptions :column="2" bordered size="small">
        <DescriptionsItem label="任务名称">{{ currentLog.taskName }}</DescriptionsItem>
        <DescriptionsItem label="任务 ID">{{ currentLog.taskId }}</DescriptionsItem>
        <DescriptionsItem label="开始时间">{{ currentLog.startTime }}</DescriptionsItem>
        <DescriptionsItem label="结束时间">{{ currentLog.endTime || '-' }}</DescriptionsItem>
        <DescriptionsItem label="触发方式">
          {{ formatTriggerMode(currentLog.triggerMode) }}
        </DescriptionsItem>
        <DescriptionsItem label="脚本来源">
          {{ formatScriptMode(currentLog.scriptMode) }}
        </DescriptionsItem>
        <DescriptionsItem label="状态">
          {{ formatStatus(currentLog.status) }}
        </DescriptionsItem>
        <DescriptionsItem label="退出码">{{ currentLog.exitCode }}</DescriptionsItem>
        <DescriptionsItem label="耗时">{{ currentLog.duration }} ms</DescriptionsItem>
        <DescriptionsItem label="输出行数">{{ outputLineCount }}</DescriptionsItem>
        <DescriptionsItem v-if="currentLog.scriptFileName" label="脚本文件">
          {{ currentLog.scriptFileName }}
        </DescriptionsItem>
        <DescriptionsItem v-if="currentLog.scriptFileVersion > 0" label="文件版本">
          v{{ currentLog.scriptFileVersion }}
        </DescriptionsItem>
        <DescriptionsItem v-if="currentLog.scriptFilePath" label="文件路径" :span="2">
          {{ currentLog.scriptFilePath }}
        </DescriptionsItem>
        <DescriptionsItem v-if="currentLog.scriptFileSha256" label="SHA256" :span="2">
          {{ currentLog.scriptFileSha256 }}
        </DescriptionsItem>
      </Descriptions>

      <div v-if="currentLog.scriptSnapshot">
        <div class="mb-2 text-sm font-medium">脚本快照</div>
        <Input.TextArea :value="currentLog.scriptSnapshot" readonly auto-size />
      </div>

      <div>
        <div class="mb-2 text-sm font-medium">标准输出</div>
        <Input.TextArea :value="currentLog.output || '-'" readonly auto-size />
      </div>

      <div v-if="currentLog.error">
        <div class="mb-2 text-sm font-medium text-red-500">错误输出</div>
        <Input.TextArea :value="currentLog.error" readonly auto-size />
      </div>
    </div>
  </Modal>
</template>
