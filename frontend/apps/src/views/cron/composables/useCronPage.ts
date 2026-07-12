import type { CronApi } from '#/api/cron';

import { computed, reactive, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import { message, Upload } from 'antdv-next';

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
import { withListDetailErrorMode } from '#/api/list-detail';
import {
  invalidateListDetailQueries,
  invalidateListDetailQueryKeys,
} from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

import { getScriptFileValidationError } from '../utils';

export function useCronPage() {
  const queryClient = useQueryClient();

  const searchName = ref('');
  const taskPagination = reactive({
    current: 1,
    pageSize: 20,
    total: 0,
  });

  const taskListParams = computed(() => ({
    name: searchName.value.trim() || undefined,
    page: taskPagination.current,
    pageSize: taskPagination.pageSize,
  }));

  const {
    data: tasksPage,
    loading,
    errorMessage: listErrorMessage,
    refetch: refetchTasks,
  } = useListDetailQuery({
    queryKey: computed(() => qk.cron.list(taskListParams.value)),
    queryFn: () =>
      getCronTaskListApi(taskListParams.value, withListDetailErrorMode()),
    errorFallback: '加载定时任务失败',
  });

  const tasks = computed(() => tasksPage.value?.list ?? []);
  watch(
    tasksPage,
    (page) => {
      taskPagination.total = page?.total ?? 0;
    },
    { immediate: true },
  );

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
  const scriptPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
  });
  const scriptFile = ref<File | null>(null);
  const scriptUploading = ref(false);
  const activatingScriptIds = ref<Set<number>>(new Set());

  const scriptHistoryParams = computed(() => ({
    taskId: currentScriptTask.value?.id ?? 0,
    page: scriptPagination.current,
    pageSize: scriptPagination.pageSize,
  }));

  const {
    data: scriptHistoryPage,
    loading: scriptHistoryLoading,
  } = useListDetailQuery({
    queryKey: computed(() =>
      qk.cron.list({
        resource: 'script-history',
        ...scriptHistoryParams.value,
      }),
    ),
    queryFn: () =>
      getCronScriptHistoryApi(
        scriptHistoryParams.value.taskId,
        {
          page: scriptHistoryParams.value.page,
          pageSize: scriptHistoryParams.value.pageSize,
        },
        withListDetailErrorMode(),
      ),
    errorFallback: '加载脚本历史失败',
    enabled: computed(() => scriptModalVisible.value && Boolean(currentScriptTask.value?.id),
    ),
  });

  const scriptHistory = computed(() => scriptHistoryPage.value?.list ?? []);
  watch(
    scriptHistoryPage,
    (page) => {
      scriptPagination.total = page?.total ?? 0;
    },
    { immediate: true },
  );

  const logDrawerVisible = ref(false);
  const logTaskId = ref<number | undefined>();
  const logPagination = reactive({
    current: 1,
    pageSize: 20,
    total: 0,
  });

  const logListParams = computed(() => ({
    page: logPagination.current,
    pageSize: logPagination.pageSize,
    taskId: logTaskId.value,
  }));

  const {
    data: logsPage,
    loading: logsLoading,
    refetch: refetchLogs,
  } = useListDetailQuery({
    queryKey: computed(() =>
      qk.cron.list({ resource: 'logs', ...logListParams.value }),
    ),
    queryFn: () =>
      getCronLogListApi(logListParams.value, withListDetailErrorMode()),
    errorFallback: '加载执行日志失败',
    enabled: computed(() => logDrawerVisible.value),
  });

  const logs = computed(() => logsPage.value?.list ?? []);
  watch(
    logsPage,
    (page) => {
      logPagination.total = page?.total ?? 0;
    },
    { immediate: true },
  );

  const logDetailVisible = ref(false);
  const currentLogId = ref<number | null>(null);

  const {
    data: currentLogData,
    loading: detailLoading,
  } = useListDetailQuery({
    queryKey: computed(() => qk.cron.detail(currentLogId.value ?? 0)),
    queryFn: () =>
      getCronLogDetailApi(currentLogId.value!, withListDetailErrorMode()),
    errorFallback: '加载日志详情失败',
    enabled: computed(() => logDetailVisible.value && currentLogId.value != null,
    ),
  });

  const currentLog = computed(() => currentLogData.value ?? null);

  const triggeringTaskIds = ref<Set<number>>(new Set());
  const deletingTaskIds = ref<Set<number>>(new Set());

  const scriptModalTitle = computed(() =>
    currentScriptTask.value
      ? `脚本管理 - ${currentScriptTask.value.name}`
      : '脚本管理',
  );
  const logDrawerTitle = computed(() =>
    logTaskId.value ? `执行日志 #${logTaskId.value}` : '全部执行日志',
  );
  const taskCanEnable = computed(() =>
      taskForm.scriptMode === 'inline' ||
      taskModalMode.value === 'edit' ||
      Boolean(taskFile.value),
  );

  function setLoadingId(
    target: typeof triggeringTaskIds,
    id: number,
    enabled: boolean,
  ) {
    const next = new Set(target.value);
    if (enabled) {
      next.add(id);
    } else {
      next.delete(id);
    }
    target.value = next;
  }

  function validateScriptFile(file: File) {
    const err = getScriptFileValidationError(file);
    if (err) {
      message.error(err);
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
    if (task.scriptMode === 'file' && task.currentFileId <= 0) {
      return '请先上传脚本';
    }
    if (task.scriptMode !== 'file' && !task.script?.trim()) {
      return '请先填写脚本';
    }
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
    await refetchTasks();
  }

  function handleSearch() {
    taskPagination.current = 1;
  }

  function handleReset() {
    searchName.value = '';
    taskPagination.current = 1;
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
    if (
      taskForm.scriptMode === 'file' &&
      taskModalMode.value === 'add' &&
      !taskFile.value
    ) {
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
      await invalidateListDetailQueries(queryClient, ['cron', 'list']);
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
      await invalidateListDetailQueries(queryClient, ['cron', 'list']);
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
      await invalidateListDetailQueries(queryClient, ['cron', 'list']);
    } catch {
      message.error('执行提交失败');
    } finally {
      setLoadingId(triggeringTaskIds, task.id, false);
    }
  }

  function handleTaskTableChange(pag: { current: number; pageSize: number }) {
    taskPagination.current = pag.current;
    taskPagination.pageSize = pag.pageSize;
  }

  function openLogs(taskId?: number) {
    logTaskId.value = taskId;
    logPagination.current = 1;
    logDrawerVisible.value = true;
  }

  async function fetchLogs() {
    await refetchLogs();
  }

  function handleLogTableChange(pag: { current: number; pageSize: number }) {
    logPagination.current = pag.current;
    logPagination.pageSize = pag.pageSize;
  }

  function openLogDetail(id: number) {
    currentLogId.value = id;
    logDetailVisible.value = true;
  }

  function openScriptManager(task: CronApi.Task) {
    currentScriptTask.value = task;
    scriptFile.value = null;
    scriptPagination.current = 1;
    scriptModalVisible.value = true;
  }

  function handleScriptTableChange(pag: {
    current: number;
    pageSize: number;
  }) {
    scriptPagination.current = pag.current;
    scriptPagination.pageSize = pag.pageSize;
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
    await invalidateListDetailQueryKeys(queryClient, [['cron', 'list']]);
    await refetchTasks();
    const synced = tasks.value.find((task) => task.id === taskId);
    if (synced) {
      currentScriptTask.value = synced;
      if (editingTask.value?.id === taskId) editingTask.value = synced;
    }
  }

  return {
    loading,
    listErrorMessage,
    tasks,
    searchName,
    taskPagination,
    taskModalVisible,
    taskModalMode,
    editingTask,
    taskSubmitting,
    taskForm,
    scriptModalVisible,
    currentScriptTask,
    scriptHistory,
    scriptHistoryLoading,
    scriptPagination,
    scriptUploading,
    activatingScriptIds,
    logDrawerVisible,
    logs,
    logsLoading,
    logPagination,
    logDetailVisible,
    detailLoading,
    currentLog,
    triggeringTaskIds,
    deletingTaskIds,
    scriptModalTitle,
    logDrawerTitle,
    taskCanEnable,
    beforeTaskFileUpload,
    beforeScriptFileUpload,
    removeTaskFile,
    removeScriptFile,
    syncFileModeStatus,
    canTriggerTask,
    fetchTasks,
    handleSearch,
    handleReset,
    openCreate,
    openEdit,
    submitTask,
    deleteTask,
    triggerTask,
    handleTaskTableChange,
    openLogs,
    fetchLogs,
    handleLogTableChange,
    openLogDetail,
    openScriptManager,
    handleScriptTableChange,
    uploadScript,
    activateScript,
  };
}
