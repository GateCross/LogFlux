<script lang="ts" setup>
import type { CronApi } from '#/api/cron';

import { computed, onMounted, reactive, ref } from 'vue';

import {
  Button,
  Card,
  Descriptions,
  DescriptionsItem,
  Drawer,
  Form,
  FormItem,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Radio,
  RadioGroup,
  Space,
  Switch,
  Table,
  Tag,
  Textarea,
  Upload,
} from 'ant-design-vue';

import {
  activateCronScriptApi,
  createCronTaskApi,
  createCronTaskWithFileApi,
  deleteCronTaskApi,
  getCronLogDetailApi,
  getCronLogListApi,
  getCronScriptHistoryApi,
  getCronTaskListApi,
  triggerCronTaskApi,
  updateCronTaskApi,
  uploadCronScriptApi,
} from '#/api/cron';

defineOptions({ name: 'CronManagement' });

const loading = ref(false);
const tasks = ref<CronApi.Task[]>([]);
const searchName = ref('');
const taskPagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
});

const taskModalVisible = ref(false);
const taskModalMode = ref<'add' | 'edit'>('add');
const editingTask = ref<CronApi.Task | null>(null);
const taskSubmitting = ref(false);
const taskFile = ref<File | null>(null);

const taskForm = reactive<CronApi.TaskPayload>({
  name: '',
  schedule: '',
  script: '',
  scriptMode: 'inline',
  status: 1,
  timeout: 60,
});

const scriptModalVisible = ref(false);
const currentScriptTask = ref<CronApi.Task | null>(null);
const scriptHistory = ref<CronApi.ScriptFile[]>([]);
const scriptHistoryLoading = ref(false);
const scriptPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
});
const scriptFile = ref<File | null>(null);
const scriptUploading = ref(false);
const activatingScriptIds = ref<Set<number>>(new Set());

const logDrawerVisible = ref(false);
const logs = ref<CronApi.Log[]>([]);
const logsLoading = ref(false);
const logTaskId = ref<number | undefined>();
const logPagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
});

const logDetailVisible = ref(false);
const detailLoading = ref(false);
const currentLog = ref<CronApi.Log | null>(null);

const triggeringTaskIds = ref<Set<number>>(new Set());
const deletingTaskIds = ref<Set<number>>(new Set());

const taskModalTitle = computed(() =>
  taskModalMode.value === 'add' ? '新增任务' : '编辑任务',
);
const scriptModalTitle = computed(() =>
  currentScriptTask.value
    ? `脚本管理 - ${currentScriptTask.value.name}`
    : '脚本管理',
);
const logDrawerTitle = computed(() =>
  logTaskId.value ? `执行日志 #${logTaskId.value}` : '全部执行日志',
);
const logDetailTitle = computed(() =>
  currentLog.value ? `日志详情 #${currentLog.value.id}` : '日志详情',
);
const outputLineCount = computed(() => {
  const output = currentLog.value?.output ?? '';
  return output.split(/\r?\n/).filter(Boolean).length;
});
const taskCanEnable = computed(
  () =>
    taskForm.scriptMode === 'inline' ||
    taskModalMode.value === 'edit' ||
    Boolean(taskFile.value),
);

const taskColumns = [
  { dataIndex: 'name', key: 'name', title: '任务名称', width: 170 },
  { dataIndex: 'schedule', key: 'schedule', title: 'Cron 表达式', width: 170 },
  { dataIndex: 'scriptMode', key: 'scriptMode', title: '脚本来源', width: 220 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 90 },
  { dataIndex: 'nextRun', key: 'nextRun', title: '下次执行', width: 170 },
  { dataIndex: 'updatedAt', key: 'updatedAt', title: '更新时间', width: 170 },
  { key: 'actions', title: '操作', width: 260 },
];

const scriptColumns = [
  { dataIndex: 'version', key: 'version', title: '版本', width: 80 },
  { dataIndex: 'originalName', key: 'originalName', title: '文件名', ellipsis: true },
  { dataIndex: 'sizeBytes', key: 'sizeBytes', title: '大小', width: 100 },
  { dataIndex: 'sha256', key: 'sha256', title: 'SHA256', ellipsis: true, width: 200 },
  { dataIndex: 'isCurrent', key: 'isCurrent', title: '状态', width: 90 },
  { dataIndex: 'createdAt', key: 'createdAt', title: '创建时间', width: 170 },
  { key: 'actions', title: '操作', width: 110 },
];

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

function setLoadingId(target: typeof triggeringTaskIds, id: number, enabled: boolean) {
  const next = new Set(target.value);
  if (enabled) {
    next.add(id);
  } else {
    next.delete(id);
  }
  target.value = next;
}

function validateScriptFile(file: File) {
  if (file.size > 1024 * 1024) {
    message.error('脚本文件不能超过 1 MiB');
    return false;
  }
  if (!file.name.toLowerCase().endsWith('.sh')) {
    message.error('仅支持上传 .sh 脚本文件');
    return false;
  }
  return true;
}

function beforeTaskFileUpload(file: File) {
  if (!validateScriptFile(file)) return Upload.LIST_IGNORE;
  taskFile.value = file;
  return false;
}

function beforeScriptFileUpload(file: File) {
  if (!validateScriptFile(file)) return Upload.LIST_IGNORE;
  scriptFile.value = file;
  return false;
}

function removeTaskFile() {
  taskFile.value = null;
  if (taskForm.scriptMode === 'file' && taskModalMode.value === 'add') {
    taskForm.status = 0;
  }
}

function removeScriptFile() {
  scriptFile.value = null;
}

function resetTaskForm() {
  Object.assign(taskForm, {
    name: '',
    schedule: '',
    script: '',
    scriptMode: 'inline',
    status: 1,
    timeout: 60,
  });
  taskFile.value = null;
}

function syncFileModeStatus() {
  if (!taskCanEnable.value) {
    taskForm.status = 0;
  }
}

function canTriggerTask(task: CronApi.Task) {
  return task.scriptMode === 'file'
    ? task.currentFileId > 0
    : Boolean(task.script?.trim());
}

function triggerTooltip(task: CronApi.Task) {
  if (task.scriptMode === 'file' && task.currentFileId <= 0) return '请先上传脚本';
  if (task.scriptMode !== 'file' && !task.script?.trim()) return '请先填写脚本';
  return '手动执行';
}

function currentTaskPayload(): CronApi.TaskPayload {
  return {
    name: taskForm.name.trim(),
    schedule: taskForm.schedule.trim(),
    script: taskForm.scriptMode === 'inline' ? taskForm.script?.trim() : '',
    scriptMode: taskForm.scriptMode,
    status: taskForm.status,
    timeout: taskForm.timeout,
  };
}

async function fetchTasks() {
  loading.value = true;
  try {
    const res = await getCronTaskListApi({
      name: searchName.value.trim() || undefined,
      page: taskPagination.current,
      pageSize: taskPagination.pageSize,
    });
    tasks.value = res.list;
    taskPagination.total = res.total;
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  taskPagination.current = 1;
  void fetchTasks();
}

function handleReset() {
  searchName.value = '';
  taskPagination.current = 1;
  void fetchTasks();
}

function openCreate() {
  taskModalMode.value = 'add';
  editingTask.value = null;
  resetTaskForm();
  taskModalVisible.value = true;
}

function openEdit(task: CronApi.Task) {
  taskModalMode.value = 'edit';
  editingTask.value = task;
  taskFile.value = null;
  Object.assign(taskForm, {
    name: task.name,
    schedule: task.schedule,
    script: task.script,
    scriptMode: task.scriptMode,
    status: task.status,
    timeout: task.timeout,
  });
  taskModalVisible.value = true;
}

async function submitTask() {
  if (!taskForm.name.trim()) {
    message.warning('请输入任务名称');
    return;
  }
  if (!taskForm.schedule.trim()) {
    message.warning('请输入 Cron 表达式');
    return;
  }
  if (taskForm.scriptMode === 'inline' && !taskForm.script?.trim()) {
    message.warning('请输入执行脚本');
    return;
  }
  if (taskForm.scriptMode === 'file' && taskModalMode.value === 'add' && !taskFile.value) {
    message.warning('请先选择脚本文件');
    return;
  }

  syncFileModeStatus();
  taskSubmitting.value = true;
  try {
    const payload = currentTaskPayload();
    if (taskModalMode.value === 'add') {
      if (payload.scriptMode === 'file' && taskFile.value) {
        await createCronTaskWithFileApi(payload, taskFile.value);
      } else {
        await createCronTaskApi(payload);
      }
    } else if (editingTask.value) {
      await updateCronTaskApi(editingTask.value.id, payload);
    }
    message.success('操作成功');
    taskModalVisible.value = false;
    await fetchTasks();
  } catch {
    message.error('操作失败');
  } finally {
    taskSubmitting.value = false;
  }
}

async function deleteTask(id: number) {
  setLoadingId(deletingTaskIds, id, true);
  try {
    await deleteCronTaskApi(id);
    message.success('删除成功');
    if (logTaskId.value === id) {
      logDrawerVisible.value = false;
      logTaskId.value = undefined;
    }
    if (currentScriptTask.value?.id === id) {
      scriptModalVisible.value = false;
    }
    await fetchTasks();
  } catch {
    message.error('删除失败');
  } finally {
    setLoadingId(deletingTaskIds, id, false);
  }
}

async function triggerTask(task: CronApi.Task) {
  if (!canTriggerTask(task)) {
    message.warning(triggerTooltip(task));
    return;
  }
  setLoadingId(triggeringTaskIds, task.id, true);
  try {
    await triggerCronTaskApi(task.id);
    message.success('执行已提交，可在执行日志查看结果');
  } catch {
    message.error('执行提交失败');
  } finally {
    setLoadingId(triggeringTaskIds, task.id, false);
  }
}

function handleTaskTableChange(pag: any) {
  taskPagination.current = pag.current;
  taskPagination.pageSize = pag.pageSize;
  void fetchTasks();
}

function openLogs(taskId?: number) {
  logTaskId.value = taskId;
  logPagination.current = 1;
  logDrawerVisible.value = true;
  void fetchLogs();
}

async function fetchLogs() {
  logsLoading.value = true;
  try {
    const res = await getCronLogListApi({
      page: logPagination.current,
      pageSize: logPagination.pageSize,
      taskId: logTaskId.value,
    });
    logs.value = res.list;
    logPagination.total = res.total;
  } finally {
    logsLoading.value = false;
  }
}

function handleLogTableChange(pag: any) {
  logPagination.current = pag.current;
  logPagination.pageSize = pag.pageSize;
  void fetchLogs();
}

async function openLogDetail(id: number) {
  detailLoading.value = true;
  logDetailVisible.value = true;
  currentLog.value = null;
  try {
    currentLog.value = await getCronLogDetailApi(id);
  } finally {
    detailLoading.value = false;
  }
}

function openScriptManager(task: CronApi.Task) {
  currentScriptTask.value = task;
  scriptFile.value = null;
  scriptPagination.current = 1;
  scriptModalVisible.value = true;
  void fetchScriptHistory();
}

async function fetchScriptHistory() {
  if (!currentScriptTask.value) return;
  scriptHistoryLoading.value = true;
  try {
    const res = await getCronScriptHistoryApi(currentScriptTask.value.id, {
      page: scriptPagination.current,
      pageSize: scriptPagination.pageSize,
    });
    scriptHistory.value = res.list;
    scriptPagination.total = res.total;
  } finally {
    scriptHistoryLoading.value = false;
  }
}

function handleScriptTableChange(pag: any) {
  scriptPagination.current = pag.current;
  scriptPagination.pageSize = pag.pageSize;
  void fetchScriptHistory();
}

async function uploadScript() {
  if (!currentScriptTask.value) return;
  if (!scriptFile.value) {
    message.warning('请先选择脚本文件');
    return;
  }

  scriptUploading.value = true;
  try {
    await uploadCronScriptApi(currentScriptTask.value.id, scriptFile.value);
    message.success('脚本上传成功');
    scriptFile.value = null;
    await refreshTaskAndScript(currentScriptTask.value.id);
  } catch {
    message.error('脚本上传失败');
  } finally {
    scriptUploading.value = false;
  }
}

async function activateScript(file: CronApi.ScriptFile) {
  if (!currentScriptTask.value) return;
  setLoadingId(activatingScriptIds, file.id, true);
  try {
    await activateCronScriptApi(currentScriptTask.value.id, file.id);
    message.success('已切换为当前脚本版本');
    await refreshTaskAndScript(currentScriptTask.value.id);
  } catch {
    message.error('激活失败');
  } finally {
    setLoadingId(activatingScriptIds, file.id, false);
  }
}

async function refreshTaskAndScript(taskId: number) {
  await fetchTasks();
  const synced = tasks.value.find((task) => task.id === taskId);
  if (synced) {
    currentScriptTask.value = synced;
    if (editingTask.value?.id === taskId) editingTask.value = synced;
  }
  await fetchScriptHistory();
}

function formatScriptMode(mode: string) {
  return mode === 'file' ? '上传脚本' : '手写脚本';
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

function taskStatusText(status: number) {
  return status === 1 ? '启用' : '禁用';
}

function statusColor(status: number) {
  switch (status) {
    case 0:
      return 'blue';
    case 1:
      return 'green';
    case 2:
      return 'red';
    case 3:
      return 'orange';
    default:
      return 'default';
  }
}

function formatTriggerMode(mode: string) {
  if (mode === 'schedule') return '定时触发';
  if (mode === 'manual') return '手动触发';
  return mode || '-';
}

function formatBytes(bytes: number) {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function shortenHash(hash: string) {
  if (!hash) return '-';
  return hash.length <= 16 ? hash : `${hash.slice(0, 8)}...${hash.slice(-8)}`;
}

onMounted(() => {
  void fetchTasks();
});
</script>

<template>
  <div class="p-5">
    <Card title="定时任务">
      <template #extra>
        <Space>
          <Button @click="openLogs()">全部日志</Button>
          <Button @click="fetchTasks">刷新</Button>
          <Button type="primary" @click="openCreate">新增任务</Button>
        </Space>
      </template>

      <div class="mb-4 flex flex-wrap items-center gap-3">
        <Input
          v-model:value="searchName"
          allow-clear
          class="w-72"
          placeholder="按任务名称搜索"
          @press-enter="handleSearch"
        />
        <Space>
          <Button type="primary" @click="handleSearch">搜索</Button>
          <Button @click="handleReset">重置</Button>
        </Space>
      </div>

      <Table
        :columns="taskColumns"
        :data-source="tasks"
        :loading="loading"
        :pagination="{
          current: taskPagination.current,
          pageSize: taskPagination.pageSize,
          total: taskPagination.total,
          showSizeChanger: true,
          showTotal: (total: number) => `共 ${total} 条`,
        }"
        row-key="id"
        size="middle"
        @change="handleTaskTableChange"
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
            <Space>
              <Button
                type="link"
                size="small"
                :disabled="!canTriggerTask(record as CronApi.Task)"
                :loading="triggeringTaskIds.has(record.id)"
                @click="triggerTask(record as CronApi.Task)"
              >
                执行
              </Button>
              <Button type="link" size="small" @click="openScriptManager(record as CronApi.Task)">
                脚本
              </Button>
              <Button type="link" size="small" @click="openLogs(record.id)">
                日志
              </Button>
              <Button type="link" size="small" @click="openEdit(record as CronApi.Task)">
                编辑
              </Button>
              <Popconfirm
                title="确认删除该任务？"
                ok-text="确认"
                cancel-text="取消"
                @confirm="deleteTask(record.id)"
              >
                <Button
                  type="link"
                  size="small"
                  danger
                  :loading="deletingTaskIds.has(record.id)"
                >
                  删除
                </Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      :confirm-loading="taskSubmitting"
      :open="taskModalVisible"
      :title="taskModalTitle"
      :width="720"
      @cancel="taskModalVisible = false"
      @ok="submitTask"
    >
      <Form :label-col="{ span: 5 }" :wrapper-col="{ span: 19 }" class="mt-4">
        <FormItem label="任务名称" required>
          <Input v-model:value="taskForm.name" />
        </FormItem>

        <FormItem label="Cron 表达式" required>
          <Input v-model:value="taskForm.schedule" placeholder="例如 0/5 * * * * ?" />
        </FormItem>

        <FormItem label="脚本来源">
          <RadioGroup v-model:value="taskForm.scriptMode" @change="syncFileModeStatus">
            <Radio value="inline">手写脚本</Radio>
            <Radio value="file">上传脚本</Radio>
          </RadioGroup>
        </FormItem>

        <FormItem v-if="taskForm.scriptMode === 'inline'" label="执行脚本" required>
          <Textarea
            v-model:value="taskForm.script"
            :rows="8"
            placeholder="请输入 Shell 脚本"
          />
        </FormItem>

        <FormItem v-else label="脚本文件" :required="taskModalMode === 'add'">
          <Upload
            v-if="taskModalMode === 'add'"
            :before-upload="beforeTaskFileUpload"
            :max-count="1"
            accept=".sh"
            @remove="removeTaskFile"
          >
            <Button>选择脚本文件</Button>
          </Upload>
          <div v-else class="text-sm text-gray-500">
            如需上传新版本，请在脚本管理中操作。
            <template v-if="editingTask?.currentFileName">
              当前脚本：v{{ editingTask.currentFileVersion }} · {{ editingTask.currentFileName }}
            </template>
          </div>
        </FormItem>

        <FormItem label="超时时间">
          <InputNumber v-model:value="taskForm.timeout" :min="1" :max="3600" class="w-full" />
        </FormItem>

        <FormItem label="启用">
          <Switch
            v-model:checked="taskForm.status"
            :checked-value="1"
            :unchecked-value="0"
            :disabled="!taskCanEnable"
          />
        </FormItem>
      </Form>
    </Modal>

    <Modal
      :open="scriptModalVisible"
      :title="scriptModalTitle"
      :width="960"
      footer=""
      @cancel="scriptModalVisible = false"
    >
      <div v-if="currentScriptTask" class="space-y-4">
        <Descriptions :column="2" bordered size="small">
          <DescriptionsItem label="任务名称">
            {{ currentScriptTask.name }}
          </DescriptionsItem>
          <DescriptionsItem label="脚本来源">
            {{ formatScriptMode(currentScriptTask.scriptMode) }}
          </DescriptionsItem>
          <DescriptionsItem label="当前脚本">
            <span v-if="currentScriptTask.currentFileName">
              v{{ currentScriptTask.currentFileVersion }} · {{ currentScriptTask.currentFileName }}
            </span>
            <span v-else>-</span>
          </DescriptionsItem>
          <DescriptionsItem label="脚本路径">
            {{ currentScriptTask.currentFilePath || '-' }}
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
              <Button type="primary" :loading="scriptUploading" @click="uploadScript">
                上传脚本
              </Button>
            </Space>
          </div>
          <div class="text-xs text-gray-500">上传后会切换为文件脚本模式，并保留历史版本。</div>
        </div>

        <Table
          :columns="scriptColumns"
          :data-source="scriptHistory"
          :loading="scriptHistoryLoading"
          :pagination="{
            current: scriptPagination.current,
            pageSize: scriptPagination.pageSize,
            total: scriptPagination.total,
            showSizeChanger: true,
          }"
          row-key="id"
          size="small"
          @change="handleScriptTableChange"
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
                type="link"
                size="small"
                :loading="activatingScriptIds.has(record.id)"
                @click="activateScript(record as CronApi.ScriptFile)"
              >
                激活
              </Button>
            </template>
          </template>
        </Table>
      </div>
    </Modal>

    <Drawer
      v-model:open="logDrawerVisible"
      :title="logDrawerTitle"
      :width="960"
      :body-style="{ padding: '16px' }"
    >
      <div class="mb-3 flex justify-end">
        <Button @click="fetchLogs">刷新</Button>
      </div>
      <Table
        :columns="logColumns"
        :data-source="logs"
        :loading="logsLoading"
        :pagination="{
          current: logPagination.current,
          pageSize: logPagination.pageSize,
          total: logPagination.total,
          showSizeChanger: true,
          showTotal: (total: number) => `共 ${total} 条`,
        }"
        row-key="id"
        size="small"
        @change="handleLogTableChange"
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
            <Button type="link" size="small" @click="openLogDetail(record.id)">
              详情
            </Button>
          </template>
        </template>
      </Table>
    </Drawer>

    <Modal
      :open="logDetailVisible"
      :title="logDetailTitle"
      :width="960"
      footer=""
      @cancel="logDetailVisible = false"
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
          <Textarea :value="currentLog.scriptSnapshot" readonly auto-size />
        </div>

        <div>
          <div class="mb-2 text-sm font-medium">标准输出</div>
          <Textarea :value="currentLog.output || '-'" readonly auto-size />
        </div>

        <div v-if="currentLog.error">
          <div class="mb-2 text-sm font-medium text-red-500">错误输出</div>
          <Textarea :value="currentLog.error" readonly auto-size />
        </div>
      </div>
    </Modal>
  </div>
</template>
