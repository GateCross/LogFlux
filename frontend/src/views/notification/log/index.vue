<template>
  <div class="h-full">
    <n-card :title="$t('page.notification.log.title')" :bordered="false" class="h-full rounded-2xl shadow-sm">
       <template #header-extra>
         <div class="flex gap-2">
            <n-select 
               v-model:value="filters.status" 
               :options="statusOptions" 
               :placeholder="$t('page.notification.log.status')" 
               clearable 
               class="w-32"
               @update:value="handleFilterChange"
            />
            <n-select
               v-model:value="filters.channelId"
               :options="channelOptions"
               :placeholder="$t('page.notification.log.channel')"
               clearable
               filterable
               class="w-40"
               @update:value="handleFilterChange"
            />
            <n-select
               v-model:value="filters.jobStatus"
               :options="jobStatusOptions"
               :placeholder="$t('page.notification.log.jobStatus')"
               clearable
               class="w-40"
               @update:value="handleFilterChange"
            />
            <n-button @click="fetchData">
               <template #icon><icon-ic-round-refresh /></template>
               {{ $t('page.notification.log.refresh') }}
            </n-button>
            <n-button
              type="error"
              :disabled="checkedRowKeys.length === 0"
              @click="handleBatchDelete"
            >
              {{ $t('common.batchDelete') }}
            </n-button>
            <n-button type="error" secondary @click="handleClear">
              {{ $t('page.notification.log.actions.clear') }}
            </n-button>
         </div>
       </template>

      <n-data-table
        remote
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        class="h-full"
        flex-height
        :row-key="row => row.id"
        :checked-row-keys="checkedRowKeys"
        @update:checked-row-keys="handleCheckedRowKeysChange"
        @update:page="handlePageChange"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, reactive, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { NButton, NTag, useDialog, useMessage } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { batchDeleteNotificationLogs, clearNotificationLogs, deleteNotificationLog, getChannelList, getLogList } from '@/service/api/notification';
import type { LogItem } from '@/service/api/notification';

const { t } = useI18n();
const message = useMessage();
const dialog = useDialog();

const loading = ref(false);
const tableData = ref<LogItem[]>([]);
const checkedRowKeys = ref<number[]>([]);
const pagination = reactive({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  onChange: (page: number) => {
    pagination.page = page;
    fetchData();
  }
});

const filters = reactive({
  status: null as number | null,
  jobStatus: null as string | null,
  channelId: null as number | null,
  ruleId: null as number | null
});

const channelOptions = ref<{label: string, value: number}[]>([]);

const statusOptions = computed(() => [
  { label: t('page.notification.log.statuses.pending'), value: 0 },
  { label: t('page.notification.log.statuses.sending'), value: 1 },
  { label: t('page.notification.log.statuses.success'), value: 2 },
  { label: t('page.notification.log.statuses.failed'), value: 3 }
]);

const jobStatusOptions = computed(() => [
  { label: t('page.notification.log.jobStatuses.queued'), value: 'queued' },
  { label: t('page.notification.log.jobStatuses.processing'), value: 'processing' },
  { label: t('page.notification.log.jobStatuses.succeeded'), value: 'succeeded' },
  { label: t('page.notification.log.jobStatuses.failed'), value: 'failed' }
]);

const columns: DataTableColumns<LogItem> = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80 },
  { title: () => t('page.notification.log.eventTitle'), key: 'title', width: 200, ellipsis: { tooltip: true } },
  { 
     title: () => t('page.notification.log.eventType'), 
     key: 'eventType', 
     width: 100,
     render(row) {
        return h(NTag, { bordered: false, type: 'info', size: 'small' }, { default: () => row.eventType });
     }
  },
  { 
     title: () => t('page.notification.log.level'), 
     key: 'level', 
     width: 80,
     render(row) {
        let type: 'default' | 'error' | 'warning' | 'info' | 'success' = 'default';
        if (row.level === 'error') type = 'error';
        else if (row.level === 'warn') type = 'warning';
        return h(NTag, { bordered: false, type, size: 'small' }, { default: () => row.level });
     }
  },
  {
     title: () => t('page.notification.log.status'),
     key: 'status',
     width: 100,
     render(row) {
        let type: 'default' | 'error' | 'warning' | 'info' | 'success' = 'default';
        let text = 'Unknown';
        switch(row.status) {
           case 0: type = 'default'; text = t('page.notification.log.statuses.pending'); break;
           case 1: type = 'info'; text = t('page.notification.log.statuses.sending'); break;
           case 2: type = 'success'; text = t('page.notification.log.statuses.success'); break;
           case 3: type = 'error'; text = t('page.notification.log.statuses.failed'); break;
        }
        return h(NTag, { bordered: false, type }, { default: () => text });
     }
  },
  {
     title: () => t('page.notification.log.job'),
     key: 'jobStatus',
     width: 140,
     render(row) {
        let type: 'default' | 'error' | 'warning' | 'info' | 'success' = 'default';
        const status = row.jobStatus;
        if (status === 'queued') type = 'default';
        else if (status === 'processing') type = 'info';
        else if (status === 'succeeded') type = 'success';
        else if (status === 'failed') type = 'error';

        const label = status ? t(`page.notification.log.jobStatuses.${status}` as any) : '-';
        const tip = status
          ? `${t('page.notification.log.jobStatus')}: ${label}\nretry=${row.jobRetryCount}, next=${row.nextRunAt || '-'}, err=${row.lastError || '-'}`
          : '';

        return h(
          NTag,
          { bordered: false, type, size: 'small', title: tip },
          { default: () => label }
        );
     }
  },
  { title: () => t('page.notification.log.sentAt'), key: 'sentAt', width: 160 },
  { title: () => t('page.notification.log.message'), key: 'message', ellipsis: { tooltip: true } }, // Content can be long
  { title: () => t('page.notification.log.error'), key: 'error', ellipsis: { tooltip: true }, render(row) { return row.error ? h('span', { class: 'text-red-500' }, row.error) : '-'; } },
  {
     title: () => t('common.action'),
     key: 'action',
     width: 100,
     render(row) {
        return h(NButton, { size: 'small', type: 'error', onClick: () => handleDelete(row) }, { default: () => t('common.delete') });
     }
  }
];

function cleanParams(params: Record<string, any>) {
  const cleaned: Record<string, any> = {};
  for (const [k, v] of Object.entries(params)) {
    if (v === null || v === undefined || v === '') continue;
    cleaned[k] = v;
  }
  return cleaned;
}

async function fetchData() {
  loading.value = true;
  try {
    const params = cleanParams({
       page: pagination.page,
       pageSize: pagination.pageSize,
       status: filters.status,
       channelId: filters.channelId,
       ruleId: filters.ruleId,
       jobStatus: filters.jobStatus
    });
    const { data, error } = await getLogList(params);
    if (!error && data) {
      tableData.value = data.list || [];
      pagination.itemCount = data.total || 0;
    }
  } finally {
    loading.value = false;
  }
}

async function fetchChannels() {
   const { data } = await getChannelList();
   if (data?.list) {
      channelOptions.value = data.list.map((c: any) => ({ label: c.name, value: c.id }));
   }
}

function handleCheckedRowKeysChange(keys: Array<number | string>) {
  checkedRowKeys.value = keys.map(k => Number(k)).filter(k => !Number.isNaN(k));
}

function handleFilterChange() {
   pagination.page = 1;
   checkedRowKeys.value = [];
   fetchData();
}

function handlePageChange(page: number) {
   pagination.page = page;
   checkedRowKeys.value = [];
   fetchData();
}

function handleDelete(row: LogItem) {
  dialog.warning({
    title: t('common.confirm'),
    content: t('page.notification.log.actions.deleteConfirm'),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const { error } = await deleteNotificationLog(row.id);
      if (!error) {
        message.success(t('common.deleteSuccess'));
        checkedRowKeys.value = checkedRowKeys.value.filter(id => id !== row.id);
        fetchData();
      } else {
        message.error(t('common.deleteFailed'));
      }
    }
  });
}

function handleBatchDelete() {
  if (checkedRowKeys.value.length === 0) return;

  dialog.warning({
    title: t('common.confirm'),
    content: t('page.notification.log.actions.batchDeleteConfirm', { count: checkedRowKeys.value.length }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const { error } = await batchDeleteNotificationLogs(checkedRowKeys.value);
      if (!error) {
        message.success(t('common.deleteSuccess'));
        checkedRowKeys.value = [];
        pagination.page = 1;
        fetchData();
      } else {
        message.error(t('common.deleteFailed'));
      }
    }
  });
}

function handleClear() {
  dialog.warning({
    title: t('common.confirm'),
    content: t('page.notification.log.actions.clearConfirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const { error } = await clearNotificationLogs();
      if (!error) {
        message.success(t('common.deleteSuccess'));
        checkedRowKeys.value = [];
        pagination.page = 1;
        fetchData();
      } else {
        message.error(t('common.deleteFailed'));
      }
    }
  });
}

onMounted(() => {
  fetchData();
  fetchChannels();
});
</script>
