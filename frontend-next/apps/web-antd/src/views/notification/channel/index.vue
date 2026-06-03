<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import { computed, onMounted, reactive, ref } from 'vue';

import {
  Button,
  Card,
  Form,
  FormItem,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  message,
} from 'ant-design-vue';

import {
  createNotificationChannelApi,
  deleteNotificationChannelApi,
  getNotificationChannelsApi,
  testNotificationChannelApi,
  updateNotificationChannelApi,
} from '#/api/notification';

defineOptions({ name: 'NotificationChannel' });

// ── List state ──────────────────────────────────────────────

const loading = ref(false);
const channels = ref<NotificationApi.Channel[]>([]);

async function fetchChannels() {
  loading.value = true;
  try {
    channels.value = await getNotificationChannelsApi();
  } finally {
    loading.value = false;
  }
}

// ── Table columns ───────────────────────────────────────────

const columns = [
  { dataIndex: 'name', key: 'name', title: 'Name' },
  { dataIndex: 'type', key: 'type', title: 'Type' },
  { dataIndex: 'status', key: 'status', title: 'Status' },
  { dataIndex: 'endpoint', key: 'endpoint', title: 'Endpoint', ellipsis: true },
  { key: 'actions', title: 'Actions', width: 240 },
];

// ── Create / Edit modal ─────────────────────────────────────

const modalVisible = ref(false);
const modalLoading = ref(false);
const editingId = ref<string | null>(null);
const isEditing = computed(() => editingId.value !== null);

const channelTypeOptions = [
  { label: 'Email', value: 'email' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'DingTalk', value: 'dingtalk' },
  { label: 'WeChat', value: 'wechat' },
  { label: 'Slack', value: 'slack' },
  { label: 'Webhook', value: 'webhook' },
];

const formState = reactive<NotificationApi.ChannelParams>({
  config: {},
  name: '',
  status: 'enabled',
  type: 'email',
});

// Dedicated fields for config sub-properties
const endpoint = ref('');
const credentials = ref('');

function openCreate() {
  editingId.value = null;
  Object.assign(formState, {
    config: {},
    name: '',
    status: 'enabled',
    type: 'email',
  });
  endpoint.value = '';
  credentials.value = '';
  modalVisible.value = true;
}

function openEdit(record: NotificationApi.Channel) {
  editingId.value = record.id;
  Object.assign(formState, {
    config: record.config,
    name: record.name,
    status: record.status,
    type: record.type,
  });
  endpoint.value = record.config?.endpoint ?? '';
  credentials.value = record.config?.credentials ?? '';
  modalVisible.value = true;
}

async function handleSubmit() {
  modalLoading.value = true;
  try {
    const payload: NotificationApi.ChannelParams = {
      ...formState,
      config: {
        ...formState.config,
        credentials: credentials.value,
        endpoint: endpoint.value,
      },
    };
    if (isEditing.value) {
      await updateNotificationChannelApi(editingId.value!, payload);
      message.success('Channel updated successfully');
    } else {
      await createNotificationChannelApi(payload);
      message.success('Channel created successfully');
    }
    modalVisible.value = false;
    await fetchChannels();
  } catch {
    message.error('Operation failed');
  } finally {
    modalLoading.value = false;
  }
}

// ── Delete ──────────────────────────────────────────────────

async function handleDelete(id: string) {
  try {
    await deleteNotificationChannelApi(id);
    message.success('Channel deleted');
    await fetchChannels();
  } catch {
    message.error('Delete failed');
  }
}

// ── Test ────────────────────────────────────────────────────

const testingId = ref<string | null>(null);

async function handleTest(id: string) {
  testingId.value = id;
  try {
    await testNotificationChannelApi({ channelId: id });
    message.success('Test notification sent successfully');
  } catch {
    message.error('Test notification failed');
  } finally {
    testingId.value = null;
  }
}

// ── Init ────────────────────────────────────────────────────

onMounted(() => {
  fetchChannels();
});
</script>

<template>
  <div class="p-5">
    <Card title="Notification Channels">
      <template #extra>
        <Button type="primary" @click="openCreate">
          Add Channel
        </Button>
      </template>

      <Table
        :columns="columns"
        :data-source="channels"
        :loading="loading"
        row-key="id"
        size="middle"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <Tag color="blue">{{ record.type }}</Tag>
          </template>

          <template v-if="column.key === 'status'">
            <Tag :color="record.status === 'enabled' ? 'green' : 'default'">
              {{ record.status === 'enabled' ? 'Enabled' : 'Disabled' }}
            </Tag>
          </template>

          <template v-if="column.key === 'endpoint'">
            {{ record.config?.endpoint ?? '-' }}
          </template>

          <template v-if="column.key === 'actions'">
            <Space>
              <Button
                size="small"
                type="link"
                :loading="testingId === record.id"
                @click="handleTest(record.id)"
              >
                Test
              </Button>
              <Button
                size="small"
                type="link"
                @click="openEdit(record)"
              >
                Edit
              </Button>
              <Popconfirm
                title="Are you sure to delete this channel?"
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
      :confirm-loading="modalLoading"
      :title="isEditing ? 'Edit Channel' : 'Add Channel'"
      :width="560"
      @ok="handleSubmit"
    >
      <Form
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
        class="mt-4"
      >
        <FormItem label="Name" required>
          <Input v-model:value="formState.name" placeholder="Channel name" />
        </FormItem>
        <FormItem label="Type" required>
          <Select
            v-model:value="formState.type"
            :options="channelTypeOptions"
            placeholder="Select channel type"
          />
        </FormItem>
        <FormItem label="Endpoint" required>
          <Input
            v-model:value="endpoint"
            placeholder="e.g. https://api.example.com or email address"
          />
        </FormItem>
        <FormItem label="Credentials">
          <Input.Password
            v-model:value="credentials"
            placeholder="API key or token"
          />
        </FormItem>
        <FormItem label="Enabled">
          <Switch
            :checked="formState.status === 'enabled'"
            @change="(val: boolean) => (formState.status = val ? 'enabled' : 'disabled')"
          />
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>
