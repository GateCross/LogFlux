<template>
  <div class="h-full flex flex-col">
    <div class="mb-4">
      <n-button @click="getData" size="small">刷新</n-button>
    </div>
    <n-data-table
      :columns="columns"
      :data="data"
      :loading="loading"
      :pagination="pagination"
      remote
      class="flex-1"
      flex-height
      @update:page="handlePageChange"
    />
    
    <!-- 日志详情弹窗 -->
    <n-modal v-model:show="showDetail" preset="card" title="日志详情" class="w-800px">
      <n-descriptions bordered :column="1" label-placement="left" :label-width="100">
        <n-descriptions-item label="任务名称">{{ currentLog?.taskName }}</n-descriptions-item>
        <n-descriptions-item label="开始时间">{{ currentLog?.startTime }}</n-descriptions-item>
        <n-descriptions-item label="结束时间">{{ currentLog?.endTime }}</n-descriptions-item>
        <n-descriptions-item label="耗时">{{ currentLog?.duration }}ms</n-descriptions-item>
        <n-descriptions-item label="Exit Code">{{ currentLog?.exitCode }}</n-descriptions-item>
      </n-descriptions>
      
      <div class="mt-4">
        <div class="font-bold mb-1">Standard Output:</div>
        <n-log :log="currentLog?.output || '(Empty)'" :rows="10" />
      </div>
      
      <div v-if="currentLog?.error" class="mt-4">
         <div class="font-bold mb-1 text-red-500">Error:</div>
         <pre class="bg-red-50 p-2 rounded text-red-600 whitespace-pre-wrap">{{ currentLog?.error }}</pre>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, watch, onMounted } from 'vue';
import { NTag, NButton, NLog, type DataTableColumns } from 'naive-ui';
import { fetchCronLogList, type CronTaskLog } from '@/service/api/cron';

const props = defineProps<{
  taskId?: number;
}>();

const data = ref<CronTaskLog[]>([]);
const loading = ref(false);
const pagination = reactive({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  onChange: (page: number) => {
    pagination.page = page;
    getData();
  }
});

const showDetail = ref(false);
const currentLog = ref<CronTaskLog | null>(null);

const columns: DataTableColumns<CronTaskLog> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '任务名称', key: 'taskName', width: 150 },
  { title: '开始时间', key: 'startTime', width: 180 },
  { title: '耗时 (ms)', key: 'duration', width: 100 },
  { 
    title: '状态', 
    key: 'status', 
    width: 100,
    render: (row) => {
      let type: 'default' | 'success' | 'warning' | 'error' | 'info' = 'default';
      let text = '未知';
      switch (row.status) {
        case 0: type = 'info'; text = '运行中'; break;
        case 1: type = 'success'; text = '成功'; break;
        case 2: type = 'error'; text = '失败'; break;
        case 3: type = 'warning'; text = '超时'; break;
      }
      return h(NTag, { type: type as any, size: 'small' }, { default: () => text });
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 100,
    render(row) {
      return h(NButton, {
        size: 'small',
        onClick: () => {
          currentLog.value = row;
          showDetail.value = true;
        }
      }, { default: () => '详情' });
    }
  }
];

async function getData() {
  loading.value = true;
  try {
    const { data: res } = await fetchCronLogList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      taskId: props.taskId
    });
    if (res) {
      data.value = res.list;
      pagination.itemCount = res.total;
    }
  } finally {
    loading.value = false;
  }
}

function handlePageChange(page: number) {
  pagination.page = page;
  getData();
}

watch(() => props.taskId, () => {
  pagination.page = 1;
  getData();
});

onMounted(() => {
  getData();
});
</script>
