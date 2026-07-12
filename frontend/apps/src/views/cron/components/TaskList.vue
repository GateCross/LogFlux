<script lang="ts" setup>
import type { CronApi } from '#/api/cron';

import { Button, Popconfirm, Space, Table, Tag } from 'antdv-next';

import {
  formatScriptMode,
  taskStatusText,
} from '../utils';

defineProps<{
  tasks: CronApi.Task[];
  loading: boolean;
  pagination: {
    current: number;
    pageSize: number;
    total: number;
  };
  triggeringTaskIds: Set<number>;
  deletingTaskIds: Set<number>;
  canTriggerTask: (task: CronApi.Task) => boolean;
}>();

const emit = defineEmits<{
  tableChange: [pag: { current: number; pageSize: number }];
  trigger: [task: CronApi.Task];
  openScript: [task: CronApi.Task];
  openLogs: [taskId: number];
  openEdit: [task: CronApi.Task];
  delete: [id: number];
}>();

const taskColumns = [
  { dataIndex: 'name', key: 'name', title: '任务名称', width: 170 },
  { dataIndex: 'schedule', key: 'schedule', title: 'Cron 表达式', width: 170 },
  { dataIndex: 'scriptMode', key: 'scriptMode', title: '脚本来源', width: 220 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 90 },
  { dataIndex: 'nextRun', key: 'nextRun', title: '下次执行', width: 170 },
  { dataIndex: 'updatedAt', key: 'updatedAt', title: '更新时间', width: 170 },
  { key: 'actions', title: '操作', width: 320 },
];

function onTableChange(pag: { current?: number; pageSize?: number }) {
  emit('tableChange', {
    current: pag.current ?? 1,
    pageSize: pag.pageSize ?? 20,
  });
}
</script>

<template>
  <div>
    <Table
      :columns="taskColumns"
      :data-source="tasks"
      :loading="loading"
      :pagination="{
        current: pagination.current,
        pageSize: pagination.pageSize,
        total: pagination.total,
        showSizeChanger: true,
        showTotal: (total: number) => `共 ${total} 条`,
      }"
      row-key="id"
      size="middle"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'scriptMode'">
          <div class="flex flex-col gap-1">
            <Tag :color="record.scriptMode === 'file' ? 'green' : 'default'">
              {{ formatScriptMode(record.scriptMode) }}
            </Tag>
            <span class="text-xs text-gray-500">
              <template v-if="record.scriptMode === 'file'">
                {{
                  record.currentFileName
                    ? `v${record.currentFileVersion} · ${record.currentFileName}`
                    : '尚未上传脚本'
                }}
              </template>
              <template v-else>
                {{ record.script ? '已配置内联脚本' : '未填写脚本' }}
              </template>
            </span>
          </div>
        </template>

        <template v-if="column.key === 'status'">
          <Tag :color="record.status === 1 ? 'green' : 'orange'">
            {{ taskStatusText(record.status) }}
          </Tag>
        </template>

        <template v-if="column.key === 'actions'">
          <Space :size="6">
            <Button
              size="small"
              class="table-action-btn table-action-btn--primary"
              :disabled="!canTriggerTask(record as CronApi.Task)"
              :loading="triggeringTaskIds.has(record.id)"
              @click="emit('trigger', record as CronApi.Task)"
            >
              执行
            </Button>
            <Button
              size="small"
              class="table-action-btn table-action-btn--secondary"
              @click="emit('openScript', record as CronApi.Task)"
            >
              脚本
            </Button>
            <Button
              size="small"
              class="table-action-btn table-action-btn--secondary"
              @click="emit('openLogs', record.id)"
            >
              日志
            </Button>
            <Button
              size="small"
              class="table-action-btn"
              @click="emit('openEdit', record as CronApi.Task)"
            >
              编辑
            </Button>
            <Popconfirm
              title="确认删除该任务？"
              ok-text="确认"
              cancel-text="取消"
              @confirm="emit('delete', record.id)"
            >
              <Button
                size="small"
                danger
                class="table-action-btn table-action-btn--danger"
                :loading="deletingTaskIds.has(record.id)"
              >
                删除
              </Button>
            </Popconfirm>
          </Space>
        </template>
      </template>
    </Table>
  </div>
</template>
