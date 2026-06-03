<script lang="ts" setup>
import type { CronApi } from '#/api/cron';

import { computed, onMounted, reactive, ref } from 'vue';

import {
  Button,
  Card,
  Drawer,
  Form,
  FormItem,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Textarea,
  message,
} from 'ant-design-vue';

import {
  createCronTaskApi,
  deleteCronTaskApi,
  getCronLogListApi,
  getCronTaskListApi,
  triggerCronTaskApi,
  updateCronTaskApi,
} from '#/api/cron';

defineOptions({ name: 'CronManagement' });

// ── Task list state ──────────────────────────────────────────

const loading = ref(false);
const tasks = ref<CronApi.Task[]>([]);

async function fetchTasks() {
  loading.value = true;
  try {
    tasks.value = await getCronTaskListApi();
  } finally {
    loading.value = false;
  }
}

// ── Table columns ────────────────────────────────────────────

const columns = [
  { dataIndex: 'name', key: 'name', title: 'Name' },
  { dataIndex: 'scriptContent', key: 'scriptContent', title: 'Script', ellipsis: true },
  { dataIndex: 'cronExpression', key: 'cronExpression', title: 'Schedule' },
  { dataIndex: 'enabled', key: 'enabled', title: 'Status' },
  { dataIndex: 'updatedAt', key: 'updatedAt', title: 'Last Updated' },
  { key: 'actions', title: 'Actions', width: 240 },
];

// ── Create / Edit modal ──────────────────────────────────────

const modalVisible = ref(false);
const editingId = ref<string | null>(null);
const isEditing = computed(() => editingId.value !== null);

const formState = reactive<CronApi.CreateTaskParams>({
  cronExpression: '',
  description: '',
  enabled: true,
  name: '',
  scriptContent: '',
  scriptType: 'inline',
  timeout: 60,
});

function openCreate() {
  editingId.value = null;
  Object.assign(formState, {
    cronExpression: '',
    description: '',
    enabled: true,
    name: '',
    scriptContent: '',
    scriptType: 'inline',
    timeout: 60,
  });
  modalVisible.value = true;
}

function openEdit(record: CronApi.Task) {
  editingId.value = record.id;
  Object.assign(formState, {
    cronExpression: record.cronExpression,
    description: record.description,
    enabled: record.enabled,
    name: record.name,
    scriptContent: record.scriptContent,
    scriptType: record.scriptType,
    timeout: record.timeout,
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  try {
    if (isEditing.value) {
      await updateCronTaskApi(editingId.value!, { ...formState });
      message.success('Task updated successfully');
    } else {
      await createCronTaskApi({ ...formState });
      message.success('Task created successfully');
    }
    modalVisible.value = false;
    await fetchTasks();
  } catch {
    message.error('Operation failed');
  }
}

// ── Delete ───────────────────────────────────────────────────

async function handleDelete(id: string) {
  try {
    await deleteCronTaskApi(id);
    message.success('Task deleted');
    await fetchTasks();
  } catch {
    message.error('Delete failed');
  }
}

// ── Trigger ──────────────────────────────────────────────────

async function handleTrigger(id: string) {
  try {
    await triggerCronTaskApi(id);
    message.success('Task triggered');
  } catch {
    message.error('Trigger failed');
  }
}

// ── Logs drawer ──────────────────────────────────────────────

const logsDrawerVisible = ref(false);
const logsLoading = ref(false);
const logList = ref<CronApi.Log[]>([]);
const logPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
});
const logFilterTaskId = ref<string>('');

const logColumns = [
  { dataIndex: 'taskName', key: 'taskName', title: 'Task' },
  { dataIndex: 'status', key: 'status', title: 'Status' },
  { dataIndex: 'triggerType', key: 'triggerType', title: 'Trigger' },
  { dataIndex: 'duration', key: 'duration', title: 'Duration (ms)' },
  { dataIndex: 'startTime', key: 'startTime', title: 'Started' },
  { dataIndex: 'endTime', key: 'endTime', title: 'Ended' },
  { dataIndex: 'errorMessage', key: 'errorMessage', title: 'Error', ellipsis: true },
];

async function openLogs(taskId?: string) {
  logFilterTaskId.value = taskId ?? '';
  logPagination.current = 1;
  logsDrawerVisible.value = true;
  await fetchLogs();
}

async function fetchLogs() {
  logsLoading.value = true;
  try {
    const res = await getCronLogListApi({
      page: logPagination.current,
      pageSize: logPagination.pageSize,
      taskId: logFilterTaskId.value || undefined,
    });
    logList.value = res.list;
    logPagination.total = res.total;
  } finally {
    logsLoading.value = false;
  }
}

function handleLogPageChange(page: number, pageSize: number) {
  logPagination.current = page;
  logPagination.pageSize = pageSize;
  fetchLogs();
}

// ── Init ─────────────────────────────────────────────────────

onMounted(() => {
  fetchTasks();
});
</script>

<template>
  <div class="p-5">
    <Card title="Cron Tasks">
      <template #extra>
        <Space>
          <Button @click="openLogs()">
            View All Logs
          </Button>
          <Button type="primary" @click="openCreate">
            Create Task
          </Button>
        </Space>
      </template>

      <Table
        :columns="columns"
        :data-source="tasks"
        :loading="loading"
        row-key="id"
        size="middle"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enabled'">
            <Tag :color="record.enabled ? 'green' : 'default'">
              {{ record.enabled ? 'Enabled' : 'Disabled' }}
            </Tag>
          </template>

          <template v-if="column.key === 'actions'">
            <Space>
              <Button
                size="small"
                type="link"
                @click="handleTrigger(record.id)"
              >
                Trigger
              </Button>
              <Button
                size="small"
                type="link"
                @click="openLogs(record.id)"
              >
                Logs
              </Button>
              <Button
                size="small"
                type="link"
                @click="openEdit(record)"
              >
                Edit
              </Button>
              <Popconfirm
                title="Are you sure to delete this task?"
                @confirm="handleDelete(record.id)"
              >
                <Button size="small" type="link" danger>
                  Delete
                </Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <!-- ── Create / Edit modal ──────────────────────────────── -->
    <Modal
      v-model:open="modalVisible"
      :title="isEditing ? 'Edit Task' : 'Create Task'"
      :width="560"
      @ok="handleSubmit"
    >
      <Form
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
        class="mt-4"
      >
        <FormItem label="Name" required>
          <Input v-model:value="formState.name" placeholder="Task name" />
        </FormItem>
        <FormItem label="Schedule" required>
          <Input
            v-model:value="formState.cronExpression"
            placeholder="Cron expression, e.g. */5 * * * *"
          />
        </FormItem>
        <FormItem label="Script Type">
          <Select v-model:value="formState.scriptType">
            <Select.Option value="inline">Inline</Select.Option>
            <Select.Option value="file">File</Select.Option>
          </Select>
        </FormItem>
        <FormItem
          v-if="formState.scriptType === 'inline'"
          label="Script"
          required
        >
          <Textarea
            v-model:value="formState.scriptContent"
            :rows="6"
            placeholder="Script content"
          />
        </FormItem>
        <FormItem label="Timeout (s)">
          <InputNumber
            v-model:value="formState.timeout"
            :min="1"
            :max="3600"
            class="w-full"
          />
        </FormItem>
        <FormItem label="Description">
          <Textarea
            v-model:value="formState.description"
            :rows="2"
            placeholder="Optional description"
          />
        </FormItem>
        <FormItem label="Enabled">
          <Switch v-model:checked="formState.enabled" />
        </FormItem>
      </Form>
    </Modal>

    <!-- ── Logs drawer ──────────────────────────────────────── -->
    <Drawer
      v-model:open="logsDrawerVisible"
      title="Execution Logs"
      :width="720"
      :body-style="{ padding: 0 }"
    >
      <div class="p-4">
        <Table
          :columns="logColumns"
          :data-source="logList"
          :loading="logsLoading"
          row-key="id"
          size="small"
          :pagination="{
            current: logPagination.current,
            pageSize: logPagination.pageSize,
            total: logPagination.total,
            showSizeChanger: true,
            showTotal: (t: number) => `Total ${t} records`,
            onChange: handleLogPageChange,
          }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <Tag
                :color="
                  record.status === 'success'
                    ? 'green'
                    : record.status === 'failed'
                      ? 'red'
                      : 'blue'
                "
              >
                {{ record.status }}
              </Tag>
            </template>
            <template v-if="column.key === 'triggerType'">
              <Tag :color="record.triggerType === 'auto' ? 'purple' : 'orange'">
                {{ record.triggerType }}
              </Tag>
            </template>
          </template>
        </Table>
      </div>
    </Drawer>
  </div>
</template>
