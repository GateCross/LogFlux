<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from 'vue';
import { type DataTableColumns, NButton, NTag, type PaginationProps } from 'naive-ui';
import { type CronLogItem, fetchCronLogDetail, fetchCronLogList } from '@/service/api/cron';
import ButtonIcon from '@/components/custom/button-icon.vue';
import SvgIcon from '@/components/custom/svg-icon.vue';

const props = defineProps<{
  taskId?: number;
}>();

const data = ref<CronLogItem[]>([]);
const loading = ref(false);
const detailLoading = ref(false);
const currentLog = ref<CronLogItem | null>(null);
const showDetail = ref(false);

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  itemCount: 0
});

const columns: DataTableColumns<CronLogItem> = [
  {
    title: 'ID',
    key: 'id',
    width: 80
  },
  {
    title: '任务名称',
    key: 'taskName',
    minWidth: 150,
    ellipsis: { tooltip: true }
  },
  {
    title: '开始时间',
    key: 'startTime',
    width: 170,
    ellipsis: { tooltip: true }
  },
  {
    title: '触发方式',
    key: 'triggerMode',
    width: 100,
    render: row =>
      h(
        NTag,
        { type: row.triggerMode === 'schedule' ? 'success' : 'info', size: 'small' },
        { default: () => formatTriggerMode(row.triggerMode) }
      )
  },
  {
    title: '脚本来源',
    key: 'scriptMode',
    width: 110,
    render: row =>
      h(
        NTag,
        { type: row.scriptMode === 'file' ? 'success' : 'default', size: 'small' },
        { default: () => formatScriptMode(row.scriptMode) }
      )
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: row =>
      h(NTag, { type: statusTagType(row.status), size: 'small' }, { default: () => formatStatus(row.status) })
  },
  {
    title: '耗时',
    key: 'duration',
    width: 100,
    render: row => `${row.duration} ms`
  },
  {
    title: '操作',
    key: 'actions',
    width: 96,
    fixed: 'right',
    render: row =>
      h(ButtonIcon, {
        icon: 'carbon:document',
        tooltipContent: '详情',
        onClick: () => handleOpenDetail(row.id)
      })
  }
];

const detailTitle = computed(() => (currentLog.value ? `日志详情 #${currentLog.value.id}` : '日志详情'));
const outputLineCount = computed(() => {
  if (!currentLog.value?.output) return 0;
  return currentLog.value.output.split(/\r?\n/).filter(Boolean).length;
});

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

async function handleOpenDetail(id: number) {
  showDetail.value = true;
  detailLoading.value = true;
  currentLog.value = null;
  try {
    const { data: res } = await fetchCronLogDetail(id);
    currentLog.value = res ?? null;
  } finally {
    detailLoading.value = false;
  }
}

function handleRefresh() {
  void getData();
}

function handlePageChange(page: number) {
  pagination.page = page;
  void getData();
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize;
  pagination.page = 1;
  void getData();
}

function formatStatus(status: number) {
  switch (status) {
    case 0:
      return '运行中';
    case 1:
      return '成功';
    case 2:
      return '失败';
    case 3:
      return '超时';
    default:
      return '未知';
  }
}

function statusTagType(status: number) {
  switch (status) {
    case 0:
      return 'info';
    case 1:
      return 'success';
    case 2:
      return 'error';
    case 3:
      return 'warning';
    default:
      return 'default';
  }
}

function formatTriggerMode(triggerMode: string) {
  return triggerMode === 'schedule' ? '定时触发' : triggerMode === 'manual' ? '手动触发' : triggerMode || '-';
}

function formatScriptMode(scriptMode: string) {
  return scriptMode === 'file' ? '上传脚本' : '手写脚本';
}

watch(
  () => props.taskId,
  () => {
    pagination.page = 1;
    void getData();
  }
);

watch(showDetail, value => {
  if (!value) {
    currentLog.value = null;
  }
});

onMounted(() => {
  void getData();
});
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="mb-3 flex items-center justify-between gap-3">
      <NText depth="3">
        {{ taskId ? '当前任务执行日志' : '执行日志' }}
      </NText>
      <NButton tertiary size="small" @click="handleRefresh">
        <template #icon>
          <SvgIcon icon="ic:round-refresh" />
        </template>
        刷新
      </NButton>
    </div>

    <NDataTable
      remote
      :columns="columns"
      :data="data"
      :loading="loading"
      :pagination="pagination"
      :row-key="row => row.id"
      class="flex-1"
      flex-height
      size="small"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />

    <NModal v-model:show="showDetail" preset="card" :title="detailTitle" class="max-h-90vh w-960px">
      <div class="max-h-78vh overflow-auto">
        <NSpin :show="detailLoading">
          <template v-if="currentLog">
            <NDescriptions bordered size="small" :column="2" label-placement="left">
              <NDescriptionsItem label="任务名称">{{ currentLog.taskName }}</NDescriptionsItem>
              <NDescriptionsItem label="任务 ID">{{ currentLog.taskId }}</NDescriptionsItem>
              <NDescriptionsItem label="开始时间">{{ currentLog.startTime }}</NDescriptionsItem>
              <NDescriptionsItem label="结束时间">{{ currentLog.endTime || '-' }}</NDescriptionsItem>
              <NDescriptionsItem label="触发方式">{{ formatTriggerMode(currentLog.triggerMode) }}</NDescriptionsItem>
              <NDescriptionsItem label="脚本来源">{{ formatScriptMode(currentLog.scriptMode) }}</NDescriptionsItem>
              <NDescriptionsItem label="状态">{{ formatStatus(currentLog.status) }}</NDescriptionsItem>
              <NDescriptionsItem label="退出码">{{ currentLog.exitCode }}</NDescriptionsItem>
              <NDescriptionsItem label="耗时">{{ currentLog.duration }} ms</NDescriptionsItem>
              <NDescriptionsItem label="输出行数">{{ outputLineCount }}</NDescriptionsItem>
              <NDescriptionsItem v-if="currentLog.scriptFileName" label="脚本文件">
                {{ currentLog.scriptFileName }}
              </NDescriptionsItem>
              <NDescriptionsItem v-if="currentLog.scriptFileVersion > 0" label="文件版本">
                v{{ currentLog.scriptFileVersion }}
              </NDescriptionsItem>
              <NDescriptionsItem v-if="currentLog.scriptFilePath" label="文件路径" :span="2">
                <NInput :value="currentLog.scriptFilePath" type="textarea" readonly autosize />
              </NDescriptionsItem>
              <NDescriptionsItem v-if="currentLog.scriptFileSha256" label="SHA256" :span="2">
                <NInput :value="currentLog.scriptFileSha256" readonly />
              </NDescriptionsItem>
              <NDescriptionsItem v-if="currentLog.scriptSnapshot" label="脚本快照" :span="2">
                <NInput :value="currentLog.scriptSnapshot" type="textarea" readonly autosize />
              </NDescriptionsItem>
            </NDescriptions>

            <div class="mt-4">
              <div class="mb-2 text-13px font-medium">标准输出</div>
              <NInput :value="currentLog.output || '-'" type="textarea" readonly autosize />
            </div>

            <div v-if="currentLog.error" class="mt-4">
              <div class="mb-2 text-13px text-error font-medium">错误输出</div>
              <NInput :value="currentLog.error" type="textarea" readonly autosize />
            </div>
          </template>
        </NSpin>
      </div>
    </NModal>
  </div>
</template>
