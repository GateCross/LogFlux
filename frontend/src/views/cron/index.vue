<template>
  <div class="h-full overflow-hidden flex flex-col">
    <n-card title="定时任务" class="h-full flex-1" content-style="display: flex; flex-direction: column; overflow: hidden;">
      <template #header-extra>
        <n-space>
          <n-button tertiary @click="handleRefresh">
            <template #icon>
              <SvgIcon icon="ic:round-refresh" />
            </template>
            刷新
          </n-button>
          <n-button type="primary" @click="handleAdd">
            <template #icon>
              <SvgIcon icon="ic:round-plus" />
            </template>
            新增任务
          </n-button>
        </n-space>
      </template>

      <div class="mb-4 flex flex-wrap items-end gap-3">
        <n-input
          v-model:value="searchName"
          clearable
          placeholder="按任务名称搜索"
          class="w-72"
          @keyup.enter="handleSearch"
        >
          <template #prefix>
            <SvgIcon icon="carbon:search" />
          </template>
        </n-input>
        <n-space>
          <n-button tertiary @click="handleSearch">搜索</n-button>
          <n-button tertiary @click="handleReset">重置</n-button>
        </n-space>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        remote
        class="flex-1"
        flex-height
        :scroll-x="1180"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <n-modal v-model:show="showTaskModal" preset="card" :title="taskModalTitle" class="w-760px">
      <n-form ref="taskFormRef" :model="taskForm" :rules="taskRules" label-placement="left" label-width="96">
        <n-grid cols="2" x-gap="16">
          <n-form-item-gi label="任务名称" path="name">
            <n-input v-model:value="taskForm.name" placeholder="请输入任务名称" />
          </n-form-item-gi>
          <n-form-item-gi label="状态" path="status">
            <n-switch v-model:value="taskForm.status" :checked-value="1" :unchecked-value="0">
              <template #checked>启用</template>
              <template #unchecked>禁用</template>
            </n-switch>
          </n-form-item-gi>
          <n-form-item-gi label="Cron 表达式" path="schedule" :span="2">
            <n-input v-model:value="taskForm.schedule" placeholder="例如：0/5 * * * * ?" />
          </n-form-item-gi>
          <n-form-item-gi label="脚本来源" path="scriptMode" :span="2">
            <n-radio-group v-model:value="taskForm.scriptMode" name="script-mode">
              <n-space>
                <n-radio-button value="inline">手写脚本</n-radio-button>
                <n-radio-button value="file">上传脚本</n-radio-button>
              </n-space>
            </n-radio-group>
          </n-form-item-gi>
          <n-form-item-gi v-if="taskForm.scriptMode === 'inline'" label="执行脚本" path="script" :span="2">
            <n-input
              v-model:value="taskForm.script"
              type="textarea"
              placeholder="请输入 Shell 脚本"
              :autosize="{ minRows: 6, maxRows: 14 }"
            />
          </n-form-item-gi>
          <n-form-item-gi v-else label="文件模式" :span="2">
            <n-alert type="info" :bordered="false">
              保存后可在脚本管理中上传脚本文件；上传后会自动切换为文件脚本模式。
            </n-alert>
            <div v-if="editingTaskInfo?.currentFileName" class="mt-3 text-12px text-neutral-500">
              当前脚本：v{{ editingTaskInfo.currentFileVersion }} · {{ editingTaskInfo.currentFileName }}
            </div>
          </n-form-item-gi>
          <n-form-item-gi label="超时时间" path="timeout">
            <n-input-number v-model:value="taskForm.timeout" :min="1" placeholder="秒" class="w-full" />
          </n-form-item-gi>
        </n-grid>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showTaskModal = false">取消</n-button>
          <n-button type="primary" :loading="submitLoading" @click="handleSubmitTask">确定</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal v-model:show="showScriptModal" preset="card" :title="scriptModalTitle" class="w-960px">
      <div v-if="currentScriptTask" class="space-y-4">
        <n-descriptions bordered size="small" :column="2" label-placement="left">
          <n-descriptions-item label="任务名称">{{ currentScriptTask.name }}</n-descriptions-item>
          <n-descriptions-item label="脚本来源">{{ currentScriptTask.scriptMode === 'file' ? '上传脚本' : '手写脚本' }}</n-descriptions-item>
          <n-descriptions-item label="当前脚本">
            <span v-if="currentScriptTask.scriptMode === 'file'">
              <span v-if="currentScriptTask.currentFileName">v{{ currentScriptTask.currentFileVersion }} · {{ currentScriptTask.currentFileName }}</span>
              <span v-else>尚未上传</span>
            </span>
            <span v-else>内联脚本</span>
          </n-descriptions-item>
          <n-descriptions-item label="脚本路径">
            <span v-if="currentScriptTask.currentFilePath">{{ currentScriptTask.currentFilePath }}</span>
            <span v-else>-</span>
          </n-descriptions-item>
        </n-descriptions>

        <div class="rounded-6px border border-neutral-200 p-4 dark:border-neutral-700">
          <div class="mb-2 flex items-center justify-between gap-3">
            <div>
              <div class="text-14px font-medium">上传新版本</div>
              <div class="text-12px text-neutral-500">仅支持 1 MiB 以内的脚本文件。</div>
            </div>
            <n-space>
              <n-button tertiary @click="handleReloadScriptHistory">
                <template #icon>
                  <SvgIcon icon="ic:round-refresh" />
                </template>
                刷新历史
              </n-button>
              <n-button type="primary" :loading="scriptUploadLoading" @click="handleUploadScript">
                <template #icon>
                  <SvgIcon icon="carbon:cloud-upload" />
                </template>
                上传脚本
              </n-button>
            </n-space>
          </div>

          <n-upload
            :key="scriptUploadKey"
            :default-upload="false"
            :max="1"
            :show-file-list="true"
            :multiple="false"
            @before-upload="handleBeforeScriptUpload"
            @remove="handleRemoveScriptUpload"
          >
            <n-button>
              <template #icon>
                <SvgIcon icon="carbon:cloud-upload" />
              </template>
              选择脚本文件
            </n-button>
          </n-upload>
          <div class="mt-2 text-12px text-neutral-500">
            上传后会切换为文件脚本模式，并保留历史版本。
          </div>
        </div>

        <n-data-table
          :columns="scriptHistoryColumns"
          :data="scriptHistoryData"
          :loading="scriptHistoryLoading"
          :pagination="scriptHistoryPagination"
          remote
          size="small"
          class="h-[360px]"
          flex-height
          :scroll-x="980"
          @update:page="handleScriptHistoryPageChange"
          @update:page-size="handleScriptHistoryPageSizeChange"
        />
      </div>

      <template #footer>
        <div class="flex justify-end">
          <n-button @click="showScriptModal = false">关闭</n-button>
        </div>
      </template>
    </n-modal>

    <n-drawer v-model:show="showLogDrawer" width="960" placement="right">
      <n-drawer-content title="执行日志">
        <cron-log-list :task-id="currentTaskId" />
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from 'vue';
import { useMessage, NButton, NPopconfirm, NRadioButton, NRadioGroup, NTag, NSpace, type DataTableColumns, type FormInst, type FormRules, type PaginationProps, type UploadFileInfo } from 'naive-ui';
import {
  activateCronTaskScript,
  createCronTask,
  deleteCronTask,
  fetchCronTaskList,
  fetchCronTaskScriptHistory,
  triggerCronTask,
  updateCronTask,
  uploadCronTaskScript,
  type CronScriptMode,
  type CronTaskFileItem,
  type CronTaskItem
} from '@/service/api/cron';
import ButtonIcon from '@/components/custom/button-icon.vue';
import SvgIcon from '@/components/custom/svg-icon.vue';
import CronLogList from './log.vue';

const message = useMessage();
const taskFormRef = ref<FormInst | null>(null);

const tableData = ref<CronTaskItem[]>([]);
const loading = ref(false);
const searchName = ref('');

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  itemCount: 0,
  onChange(page: number) {
    pagination.page = page;
    void getData();
  },
  onUpdatePageSize(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.page = 1;
    void getData();
  }
});

const showTaskModal = ref(false);
const taskModalMode = ref<'add' | 'edit'>('add');
const submitLoading = ref(false);
const editingTaskInfo = ref<CronTaskItem | null>(null);

const taskForm = reactive({
  id: 0,
  name: '',
  schedule: '',
  scriptMode: 'inline' as CronScriptMode,
  script: '',
  status: 1,
  timeout: 60
});

const taskRules = computed<FormRules>(() => ({
  name: [{ required: true, message: '请输入任务名称', trigger: ['blur', 'input'] }],
  schedule: [{ required: true, message: '请输入 Cron 表达式', trigger: ['blur', 'input'] }],
  script:
    taskForm.scriptMode === 'inline'
      ? [
          {
            validator: (_rule, value) => (String(value || '').trim() ? true : new Error('请输入执行脚本')),
            trigger: ['blur', 'input']
          }
        ]
      : [],
  timeout: [
    {
      validator: (_rule, value) => (Number(value) > 0 ? true : new Error('请输入大于 0 的超时时间')),
      trigger: ['blur', 'change']
    }
  ]
}));

const showScriptModal = ref(false);
const currentScriptTask = ref<CronTaskItem | null>(null);
const scriptHistoryData = ref<CronTaskFileItem[]>([]);
const scriptHistoryLoading = ref(false);
const scriptUploadLoading = ref(false);
const scriptUploadFile = ref<File | null>(null);
const scriptUploadKey = ref(0);

const scriptHistoryPagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 10,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  itemCount: 0,
  onChange(page: number) {
    scriptHistoryPagination.page = page;
    void loadScriptHistory();
  },
  onUpdatePageSize(pageSize: number) {
    scriptHistoryPagination.pageSize = pageSize;
    scriptHistoryPagination.page = 1;
    void loadScriptHistory();
  }
});

const showLogDrawer = ref(false);
const currentTaskId = ref(0);

const columns: DataTableColumns<CronTaskItem> = [
  {
    title: '任务名称',
    key: 'name',
    width: 170,
    ellipsis: { tooltip: true }
  },
  {
    title: 'Cron 表达式',
    key: 'schedule',
    width: 180,
    render: row => h(NTag, { type: 'info', size: 'small' }, { default: () => row.schedule })
  },
  {
    title: '脚本来源',
    key: 'scriptMode',
    minWidth: 220,
    render: row =>
      h('div', { class: 'flex flex-col gap-1' }, [
        h(
          NTag,
          { type: row.scriptMode === 'file' ? 'success' : 'default', size: 'small' },
          { default: () => formatScriptMode(row.scriptMode) }
        ),
        h(
          'span',
          { class: 'text-12px text-neutral-500' },
          row.scriptMode === 'file'
            ? row.currentFileName
              ? `v${row.currentFileVersion} · ${row.currentFileName}`
              : '尚未上传脚本'
            : row.script
              ? '已配置内联脚本'
              : '未填写脚本'
        )
      ])
  },
  {
    title: '状态',
    key: 'status',
    width: 96,
    render: row =>
      h(
        NTag,
        { type: row.status === 1 ? 'success' : 'warning', size: 'small' },
        { default: () => (row.status === 1 ? '启用' : '禁用') }
      )
  },
  {
    title: '下次执行',
    key: 'nextRun',
    width: 180,
    ellipsis: { tooltip: true }
  },
  {
    title: '更新时间',
    key: 'updatedAt',
    width: 170,
    ellipsis: { tooltip: true }
  },
  {
    title: '操作',
    key: 'actions',
    width: 210,
    fixed: 'right',
    render: row =>
      h(NSpace, { size: 6 }, {
        default: () => [
          h(ButtonIcon, {
            icon: 'carbon:play',
            tooltipContent: '手动执行',
            onClick: () => handleTrigger(row)
          }),
          h(ButtonIcon, {
            icon: 'carbon:script',
            tooltipContent: '脚本管理',
            onClick: () => openScriptManager(row)
          }),
          h(ButtonIcon, {
            icon: 'carbon:edit',
            tooltipContent: '编辑',
            onClick: () => handleEdit(row)
          }),
          h(ButtonIcon, {
            icon: 'carbon:list',
            tooltipContent: '执行日志',
            onClick: () => openLogDrawer(row.id)
          }),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => handleDelete(row.id)
            },
            {
              trigger: () =>
                h(ButtonIcon, {
                  icon: 'carbon:trash-can',
                  tooltipContent: '删除'
                }),
              default: () => '确认删除该任务吗？'
            }
          )
        ]
      })
  }
];

const scriptHistoryColumns: DataTableColumns<CronTaskFileItem> = [
  {
    title: '版本',
    key: 'version',
    width: 90,
    render: row => `v${row.version}`
  },
  {
    title: '文件名',
    key: 'originalName',
    minWidth: 180,
    ellipsis: { tooltip: true }
  },
  {
    title: '大小',
    key: 'sizeBytes',
    width: 100,
    render: row => formatBytes(row.sizeBytes)
  },
  {
    title: 'SHA256',
    key: 'sha256',
    minWidth: 180,
    ellipsis: { tooltip: true },
    render: row => shortenHash(row.sha256)
  },
  {
    title: '状态',
    key: 'isCurrent',
    width: 90,
    render: row =>
      h(
        NTag,
        { type: row.isCurrent ? 'success' : 'default', size: 'small' },
        { default: () => (row.isCurrent ? '当前' : '历史') }
      )
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 170,
    ellipsis: { tooltip: true }
  },
  {
    title: '操作',
    key: 'actions',
    width: 120,
    render: row =>
      row.isCurrent
        ? h(NTag, { type: 'success', size: 'small' }, { default: () => '已激活' })
        : h(ButtonIcon, {
            icon: 'carbon:checkmark',
            tooltipContent: '激活此版本',
            onClick: () => handleActivateScript(row)
          })
  }
];

const taskModalTitle = computed(() => (taskModalMode.value === 'add' ? '新增任务' : '编辑任务'));
const scriptModalTitle = computed(() =>
  currentScriptTask.value ? `脚本管理 - ${currentScriptTask.value.name}` : '脚本管理'
);

async function getData() {
  loading.value = true;
  try {
    const { data: res } = await fetchCronTaskList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      name: searchName.value.trim() || undefined
    });
    if (res) {
      tableData.value = res.list;
      pagination.itemCount = res.total;
    }
  } finally {
    loading.value = false;
  }
}

function handleRefresh() {
  void getData();
}

function handleSearch() {
  pagination.page = 1;
  void getData();
}

function handleReset() {
  searchName.value = '';
  pagination.page = 1;
  void getData();
}

function handleAdd() {
  taskModalMode.value = 'add';
  editingTaskInfo.value = null;
  resetTaskForm();
  showTaskModal.value = true;
}

function handleEdit(row: CronTaskItem) {
  taskModalMode.value = 'edit';
  editingTaskInfo.value = row;
  Object.assign(taskForm, {
    id: row.id,
    name: row.name,
    schedule: row.schedule,
    scriptMode: row.scriptMode,
    script: row.script,
    status: row.status,
    timeout: row.timeout
  });
  showTaskModal.value = true;
}

function resetTaskForm() {
  Object.assign(taskForm, {
    id: 0,
    name: '',
    schedule: '',
    scriptMode: 'inline' as CronScriptMode,
    script: '',
    status: 1,
    timeout: 60
  });
}

async function handleSubmitTask() {
  await taskFormRef.value?.validate();
  submitLoading.value = true;
  try {
    const payload = {
      name: taskForm.name.trim(),
      schedule: taskForm.schedule.trim(),
      scriptMode: taskForm.scriptMode,
      script: taskForm.script.trim(),
      status: taskForm.status,
      timeout: taskForm.timeout
    };

    if (taskModalMode.value === 'add') {
      const { error } = await createCronTask(payload);
      if (!error) {
        message.success('创建成功');
        showTaskModal.value = false;
        handleRefresh();
        if (taskForm.scriptMode === 'file') {
          message.info('任务已创建，请到脚本管理中上传脚本文件');
        }
      }
      return;
    }

    const { error } = await updateCronTask(taskForm.id, payload);
    if (!error) {
      message.success('更新成功');
      showTaskModal.value = false;
      handleRefresh();
    }
  } finally {
    submitLoading.value = false;
  }
}

async function handleDelete(id: number) {
  const { error } = await deleteCronTask(id);
  if (!error) {
    message.success('删除成功');
    if (currentScriptTask.value?.id === id) {
      showScriptModal.value = false;
    }
    if (currentTaskId.value === id) {
      showLogDrawer.value = false;
      currentTaskId.value = 0;
    }
    handleRefresh();
  }
}

async function handleTrigger(row: CronTaskItem) {
  const { error } = await triggerCronTask(row.id);
  if (!error) {
    message.success('已触发手动执行');
  }
}

function openLogDrawer(taskId: number) {
  currentTaskId.value = taskId;
  showLogDrawer.value = true;
}

function openScriptManager(row: CronTaskItem) {
  currentScriptTask.value = row;
  showScriptModal.value = true;
  scriptHistoryPagination.page = 1;
  resetScriptUpload();
  void loadScriptHistory();
}

async function loadScriptHistory() {
  if (!currentScriptTask.value) return;
  scriptHistoryLoading.value = true;
  try {
    const { data: res } = await fetchCronTaskScriptHistory(currentScriptTask.value.id, {
      page: scriptHistoryPagination.page,
      pageSize: scriptHistoryPagination.pageSize
    });
    if (res) {
      scriptHistoryData.value = res.list;
      scriptHistoryPagination.itemCount = res.total;
    }
  } finally {
    scriptHistoryLoading.value = false;
  }
}

function handleReloadScriptHistory() {
  void loadScriptHistory();
}

function handleBeforeScriptUpload(data: { file: UploadFileInfo }) {
  const raw = data.file.file;
  if (!raw) return false;
  if (raw.size > 1024 * 1024) {
    message.error('脚本文件不能超过 1 MiB');
    return false;
  }
  scriptUploadFile.value = raw;
  return false;
}

function handleRemoveScriptUpload() {
  scriptUploadFile.value = null;
  return true;
}

function resetScriptUpload() {
  scriptUploadFile.value = null;
  scriptUploadKey.value += 1;
}

async function handleUploadScript() {
  if (!currentScriptTask.value) return;
  if (!scriptUploadFile.value) {
    message.error('请先选择脚本文件');
    return;
  }

  scriptUploadLoading.value = true;
  try {
    const formData = new FormData();
    formData.append('file', scriptUploadFile.value);

    const { error } = await uploadCronTaskScript(currentScriptTask.value.id, formData);
    if (!error) {
      message.success('脚本上传成功');
      resetScriptUpload();
      await handleRefreshAndSyncTask(currentScriptTask.value.id);
      await loadScriptHistory();
    }
  } finally {
    scriptUploadLoading.value = false;
  }
}

async function handleActivateScript(row: CronTaskFileItem) {
  if (!currentScriptTask.value) return;
  const { error } = await activateCronTaskScript(currentScriptTask.value.id, row.id);
  if (!error) {
    message.success('已切换为当前脚本版本');
    await handleRefreshAndSyncTask(currentScriptTask.value.id);
    await loadScriptHistory();
  }
}

async function handleRefreshAndSyncTask(taskId: number) {
  await getData();
  const syncedTask = tableData.value.find(item => item.id === taskId);
  if (syncedTask) {
    currentScriptTask.value = syncedTask;
    editingTaskInfo.value = editingTaskInfo.value?.id === taskId ? syncedTask : editingTaskInfo.value;
  }
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

function handleScriptHistoryPageChange(page: number) {
  scriptHistoryPagination.page = page;
  void loadScriptHistory();
}

function handleScriptHistoryPageSizeChange(pageSize: number) {
  scriptHistoryPagination.pageSize = pageSize;
  scriptHistoryPagination.page = 1;
  void loadScriptHistory();
}

function formatScriptMode(mode: string) {
  return mode === 'file' ? '上传脚本' : '手写脚本';
}

function formatBytes(bytes: number) {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let size = bytes;
  let index = 0;
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024;
    index += 1;
  }
  return `${size.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function shortenHash(hash: string) {
  if (!hash) return '-';
  if (hash.length <= 16) return hash;
  return `${hash.slice(0, 8)}…${hash.slice(-8)}`;
}

watch(showScriptModal, value => {
  if (!value) {
    currentScriptTask.value = null;
    scriptHistoryData.value = [];
    scriptHistoryPagination.page = 1;
    resetScriptUpload();
  }
});

onMounted(() => {
  void getData();
});
</script>
