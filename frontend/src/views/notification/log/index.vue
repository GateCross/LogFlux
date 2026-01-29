<template>
  <div class="h-full">
    <n-card title="Notification Logs" :bordered="false" class="h-full rounded-2xl shadow-sm">
       <template #header-extra>
         <div class="flex gap-2">
            <n-select 
               v-model:value="filters.status" 
               :options="statusOptions" 
               placeholder="Status" 
               clearable 
               class="w-32"
               @update:value="handleFilterChange"
            />
            <n-select 
               v-model:value="filters.channelId" 
               :options="channelOptions" 
               placeholder="Channel" 
               clearable 
               filterable
               class="w-40"
               @update:value="handleFilterChange"
            />
            <n-button @click="fetchData">
               <template #icon><div class="i-carbon-renew" /></template>
               Refresh
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
        @update:page="handlePageChange"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, reactive } from 'vue';
import { NTag } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { getLogList, getChannelList } from '@/service/api/notification';
import type { LogItem } from '@/service/api/notification';

const loading = ref(false);
const tableData = ref<LogItem[]>([]);
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
  channelId: null as number | null,
  ruleId: null as number | null
});

const channelOptions = ref<{label: string, value: number}[]>([]);

const statusOptions = [
  { label: 'Pending', value: 0 },
  { label: 'Sending', value: 1 },
  { label: 'Success', value: 2 },
  { label: 'Failed', value: 3 }
];

const columns: DataTableColumns<LogItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: 'Title', key: 'title', width: 200, ellipsis: { tooltip: true } },
  { 
     title: 'Type', 
     key: 'eventType', 
     width: 100,
     render(row) {
        return h(NTag, { bordered: false, type: 'info', size: 'small' }, { default: () => row.eventType });
     }
  },
  { 
     title: 'Level', 
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
     title: 'Status', 
     key: 'status',
     width: 100,
     render(row) {
        let type: 'default' | 'error' | 'warning' | 'info' | 'success' = 'default';
        let text = 'Unknown';
        switch(row.status) {
           case 0: type = 'default'; text = 'Pending'; break;
           case 1: type = 'info'; text = 'Sending'; break;
           case 2: type = 'success'; text = 'Success'; break;
           case 3: type = 'error'; text = 'Failed'; break;
        }
        return h(NTag, { bordered: false, type }, { default: () => text });
     }
  },
  { title: 'Sent At', key: 'sentAt', width: 160 },
  { title: 'Message', key: 'message', ellipsis: { tooltip: true } }, // Content can be long
  { title: 'Error', key: 'error', ellipsis: { tooltip: true }, render(row) { return row.error ? h('span', { class: 'text-red-500' }, row.error) : '-'; } }
];

async function fetchData() {
  loading.value = true;
  try {
    const params = {
       page: pagination.page,
       pageSize: pagination.pageSize,
       status: filters.status,
       channelId: filters.channelId,
       ruleId: filters.ruleId
    };
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

function handleFilterChange() {
   pagination.page = 1;
   fetchData();
}

function handlePageChange(page: number) {
   pagination.page = page;
   fetchData();
}

onMounted(() => {
  fetchData();
  fetchChannels();
});
</script>
