<script lang="ts" setup>
import type { CronApi } from '#/api/cron';

import {
  Button,
  Descriptions,
  DescriptionsItem,
  Modal,
  Space,
  Table,
  Tag,
  Upload,
} from 'antdv-next';

import {
  formatBytes,
  formatScriptMode,
  shortenHash,
} from '../utils';

defineProps<{
  open: boolean;
  title: string;
  currentTask: CronApi.Task | null;
  scriptHistory: CronApi.ScriptFile[];
  historyLoading: boolean;
  pagination: {
    current: number;
    pageSize: number;
    total: number;
  };
  scriptUploading: boolean;
  activatingScriptIds: Set<number>;
  beforeScriptFileUpload: (file: File) => boolean | typeof Upload.LIST_IGNORE;
  removeScriptFile: () => void;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  upload: [];
  activate: [file: CronApi.ScriptFile];
  tableChange: [pag: { current: number; pageSize: number }];
}>();

const scriptColumns = [
  { dataIndex: 'version', key: 'version', title: '版本', width: 80 },
  {
    dataIndex: 'originalName',
    key: 'originalName',
    title: '文件名',
    ellipsis: true,
  },
  { dataIndex: 'sizeBytes', key: 'sizeBytes', title: '大小', width: 100 },
  {
    dataIndex: 'sha256',
    key: 'sha256',
    title: 'SHA256',
    ellipsis: true,
    width: 200,
  },
  { dataIndex: 'isCurrent', key: 'isCurrent', title: '状态', width: 90 },
  { dataIndex: 'createdAt', key: 'createdAt', title: '创建时间', width: 170 },
  { key: 'actions', title: '操作', width: 110 },
];

function onTableChange(pag: { current?: number; pageSize?: number }) {
  emit('tableChange', {
    current: pag.current ?? 1,
    pageSize: pag.pageSize ?? 10,
  });
}
</script>

<template>
  <Modal
    :open="open"
    :title="title"
    :width="960"
    footer=""
    @cancel="emit('update:open', false)"
  >
    <div v-if="currentTask" class="space-y-4">
      <Descriptions :column="2" bordered size="small">
        <DescriptionsItem label="任务名称">
          {{ currentTask.name }}
        </DescriptionsItem>
        <DescriptionsItem label="脚本来源">
          {{ formatScriptMode(currentTask.scriptMode) }}
        </DescriptionsItem>
        <DescriptionsItem label="当前脚本">
          <span v-if="currentTask.currentFileName">
            v{{ currentTask.currentFileVersion }} · {{ currentTask.currentFileName }}
          </span>
          <span v-else>-</span>
        </DescriptionsItem>
        <DescriptionsItem label="脚本路径">
          {{ currentTask.currentFilePath || '-' }}
        </DescriptionsItem>
      </Descriptions>

      <div class="rounded border border-gray-200 p-4">
        <div class="mb-3 flex items-center justify-between gap-3">
          <div>
            <div class="font-medium">上传新版本</div>
            <div class="text-xs text-gray-500">仅支持 1 MiB 以内的 .sh 脚本文件。</div>
          </div>
          <Space>
            <Upload
              :before-upload="beforeScriptFileUpload"
              :max-count="1"
              accept=".sh"
              @remove="removeScriptFile"
            >
              <Button>选择脚本文件</Button>
            </Upload>
            <Button type="primary" :loading="scriptUploading" @click="emit('upload')">
              上传脚本
            </Button>
          </Space>
        </div>
        <div class="text-xs text-gray-500">上传后会切换为文件脚本模式，并保留历史版本。</div>
      </div>

      <Table
        :columns="scriptColumns"
        :data-source="scriptHistory"
        :loading="historyLoading"
        :pagination="{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
        }"
        row-key="id"
        size="small"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'version'">
            v{{ record.version }}
          </template>
          <template v-if="column.key === 'sizeBytes'">
            {{ formatBytes(record.sizeBytes) }}
          </template>
          <template v-if="column.key === 'sha256'">
            {{ shortenHash(record.sha256) }}
          </template>
          <template v-if="column.key === 'isCurrent'">
            <Tag :color="record.isCurrent ? 'green' : 'default'">
              {{ record.isCurrent ? '当前' : '历史' }}
            </Tag>
          </template>
          <template v-if="column.key === 'actions'">
            <Tag v-if="record.isCurrent" color="green">已激活</Tag>
            <Button
              v-else
              size="small"
              class="table-action-btn table-action-btn--primary"
              :loading="activatingScriptIds.has(record.id)"
              @click="emit('activate', record as CronApi.ScriptFile)"
            >
              激活
            </Button>
          </template>
        </template>
      </Table>
    </div>
  </Modal>
</template>
