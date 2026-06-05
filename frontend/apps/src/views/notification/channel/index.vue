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

type WebhookBodySource =
  | 'content'
  | 'custom'
  | 'data'
  | 'level'
  | 'message'
  | 'timestamp'
  | 'title'
  | 'type';

interface ChannelFormState {
  config: string;
  description: string;
  enabled: boolean;
  events: string;
  name: string;
  type: string;
}

interface WebhookBodyFieldItem {
  customValue: string;
  key: string;
  source: WebhookBodySource;
}

interface WebhookConfigForm {
  body_fields: WebhookBodyFieldItem[];
  headers: WebhookHeaderItem[];
  method: string;
  url: string;
}

interface WebhookHeaderItem {
  key: string;
  value: string;
}

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

// ── Options ─────────────────────────────────────────────────

const channelTypeOptions = [
  { label: 'Webhook', value: 'webhook' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'Slack', value: 'slack' },
  { label: '企业微信机器人', value: 'wecom' },
  { label: '企业微信应用消息', value: 'wechat_mp' },
  { label: 'Discord', value: 'discord' },
  { label: '邮件', value: 'email' },
  { label: '站内通知', value: 'in_app' },
];

const methodOptions = [
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'GET', value: 'GET' },
];

const bodySourceOptions = [
  { label: '标题', value: 'title' },
  { label: '内容', value: 'content' },
  { label: '消息', value: 'message' },
  { label: '级别', value: 'level' },
  { label: '类型', value: 'type' },
  { label: '时间', value: 'timestamp' },
  { label: '事件数据', value: 'data' },
  { label: '自定义', value: 'custom' },
];

const eventOptions = [
  { label: '全部事件', value: '*' },
  { label: '系统事件', value: 'system.*' },
  { label: 'Caddy 事件', value: 'caddy.*' },
  { label: '安全事件', value: 'security.*' },
  { label: '任务事件', value: 'task.*' },
  { label: '归档事件', value: 'archive.*' },
];

function channelTypeLabel(type: string) {
  return channelTypeOptions.find((item) => item.value === type)?.label ?? type;
}

// ── Table columns ───────────────────────────────────────────

const columns = [
  { dataIndex: 'name', key: 'name', title: '名称' },
  { dataIndex: 'type', key: 'type', title: '类型', width: 150 },
  { dataIndex: 'enabled', key: 'enabled', title: '状态', width: 100 },
  { key: 'endpoint', title: '地址', ellipsis: true },
  { dataIndex: 'description', key: 'description', title: '描述', ellipsis: true },
  { key: 'actions', title: '操作', width: 240 },
];

function channelEndpoint(record: Partial<NotificationApi.Channel>) {
  const config = parseConfig(record.config);
  return config.url || config.endpoint || config.webhook || config.address || '-';
}

// ── 创建 / 编辑 modal ─────────────────────────────────────

const modalVisible = ref(false);
const modalLoading = ref(false);
const editingId = ref<null | string>(null);
const isEditing = computed(() => editingId.value !== null);

const formState = reactive<ChannelFormState>(createDefaultFormState());
const webhookForm = ref<WebhookConfigForm>(createDefaultWebhookForm());
const eventTags = ref<string[]>(['*']);
const isWebhookType = computed(() => formState.type === 'webhook');
const webhookConfigPreview = computed(() =>
  JSON.stringify(buildWebhookConfig(), null, 2),
);

function createDefaultFormState(): ChannelFormState {
  return {
    config: '{}',
    description: '',
    enabled: true,
    events: '["*"]',
    name: '',
    type: 'webhook',
  };
}

function createHeaderItem(): WebhookHeaderItem {
  return { key: '', value: '' };
}

function createBodyFieldItem(): WebhookBodyFieldItem {
  return { customValue: '', key: '', source: 'custom' };
}

function createDefaultWebhookForm(): WebhookConfigForm {
  return {
    body_fields: [
      { customValue: '', key: 'title', source: 'title' },
      { customValue: '', key: 'content', source: 'content' },
    ],
    headers: [
      { key: 'Content-Type', value: 'application/json' },
      { key: 'apiKey', value: '' },
    ],
    method: 'POST',
    url: '',
  };
}

function parseConfig(value: Record<string, any> | string | undefined) {
  if (!value) return {};
  if (typeof value !== 'string') return value;
  try {
    return JSON.parse(value);
  } catch {
    return {};
  }
}

function stringifyConfig(value: Record<string, any> | string | undefined) {
  if (!value) return '{}';
  if (typeof value === 'string') return value || '{}';
  return JSON.stringify(value, null, 2);
}

function buildWebhookConfig() {
  const headers = webhookForm.value.headers.reduce<Record<string, string>>(
    (acc, item) => {
      const key = item.key.trim();
      if (key) acc[key] = item.value.trim();
      return acc;
    },
    {},
  );

  const bodyFields = webhookForm.value.body_fields.reduce<Record<string, string>>(
    (acc, item) => {
      const key = item.key.trim();
      if (!key) return acc;
      acc[key] = item.source === 'custom' ? item.customValue : item.source;
      return acc;
    },
    {},
  );

  return {
    body_fields: bodyFields,
    headers,
    method: webhookForm.value.method,
    payload_mode: 'message_api',
    url: webhookForm.value.url.trim(),
  };
}

function applyWebhookConfig(config: Record<string, any> | string | undefined) {
  const next = createDefaultWebhookForm();
  const parsed = parseConfig(config);
  const knownSources = new Set<WebhookBodySource>([
    'content',
    'data',
    'level',
    'message',
    'timestamp',
    'title',
    'type',
  ]);

  const headers =
    parsed.headers && typeof parsed.headers === 'object'
      ? Object.entries(parsed.headers as Record<string, any>).map(([key, value]) => ({
          key,
          value: String(value ?? ''),
        }))
      : next.headers;

  const bodyFields =
    parsed.body_fields && typeof parsed.body_fields === 'object'
      ? Object.entries(parsed.body_fields as Record<string, any>).map(
          ([key, value]) => {
            const source = String(value ?? '');
            const isKnownSource = knownSources.has(source as WebhookBodySource);
            return {
              customValue: isKnownSource ? '' : source,
              key,
              source: isKnownSource ? (source as WebhookBodySource) : 'custom',
            };
          },
        )
      : next.body_fields;

  webhookForm.value = {
    ...next,
    body_fields: bodyFields.length > 0 ? bodyFields : next.body_fields,
    headers: headers.length > 0 ? headers : next.headers,
    method: typeof parsed.method === 'string' ? parsed.method : next.method,
    url: typeof parsed.url === 'string' ? parsed.url : next.url,
  };
}

function applyEventTags(eventsText: string | undefined) {
  if (!eventsText?.trim()) {
    eventTags.value = ['*'];
    return;
  }

  try {
    const parsed = JSON.parse(eventsText);
    if (Array.isArray(parsed)) {
      const tags = parsed.map((item) => String(item)).filter(Boolean);
      eventTags.value = tags.length > 0 ? tags : ['*'];
      return;
    }
  } catch {
    // fallback below
  }

  eventTags.value = ['*'];
}

function addHeader() {
  webhookForm.value.headers.push(createHeaderItem());
}

function removeHeader(index: number) {
  webhookForm.value.headers.splice(index, 1);
}

function addBodyField() {
  webhookForm.value.body_fields.push(createBodyFieldItem());
}

function removeBodyField(index: number) {
  webhookForm.value.body_fields.splice(index, 1);
}

function resetFormForCreate() {
  Object.assign(formState, createDefaultFormState());
  webhookForm.value = createDefaultWebhookForm();
  eventTags.value = ['*'];
}

function openCreate() {
  editingId.value = null;
  resetFormForCreate();
  modalVisible.value = true;
}

function openEdit(record: NotificationApi.Channel) {
  editingId.value = String(record.id);
  Object.assign(formState, {
    config: stringifyConfig(record.config),
    description: record.description ?? '',
    enabled: record.enabled ?? record.status !== 'disabled',
    events: record.events ?? '["*"]',
    name: record.name,
    type: record.type,
  });
  applyEventTags(formState.events);
  applyWebhookConfig(record.config);
  modalVisible.value = true;
}

function handleTypeChange(value: string) {
  formState.type = value;
  if (value === 'webhook') {
    applyWebhookConfig(formState.config);
  }
}

function validateChannelForm() {
  if (!formState.name.trim()) {
    message.warning('请输入渠道名称');
    return false;
  }
  if (!formState.type) {
    message.warning('请选择渠道类型');
    return false;
  }
  if (eventTags.value.length === 0) {
    message.warning('请至少填写一个事件范围');
    return false;
  }
  if (isWebhookType.value) {
    if (!webhookForm.value.url.trim()) {
      message.warning('请输入 Webhook 地址');
      return false;
    }
    const hasBodyField = webhookForm.value.body_fields.some((item) =>
      item.key.trim(),
    );
    if (!hasBodyField) {
      message.warning('请至少填写一个请求正文字段');
      return false;
    }
    return true;
  }

  if (!formState.config.trim()) {
    formState.config = '{}';
    return true;
  }
  try {
    JSON.parse(formState.config);
    return true;
  } catch {
    message.warning('配置必须是合法的 JSON');
    return false;
  }
}

async function handleSubmit() {
  if (!validateChannelForm()) return;

  modalLoading.value = true;
  try {
    const config = isWebhookType.value
      ? buildWebhookConfig()
      : parseConfig(formState.config);
    const payload: NotificationApi.ChannelParams = {
      config,
      description: formState.description,
      enabled: formState.enabled,
      events: JSON.stringify(eventTags.value),
      name: formState.name,
      status: formState.enabled ? 'enabled' : 'disabled',
      type: formState.type,
    };

    if (isEditing.value) {
      await updateNotificationChannelApi(editingId.value!, payload);
      message.success('通知渠道已更新');
    } else {
      await createNotificationChannelApi(payload);
      message.success('通知渠道已创建');
    }
    modalVisible.value = false;
    await fetchChannels();
  } catch {
    message.error('操作失败');
  } finally {
    modalLoading.value = false;
  }
}

// ── 删除 ──────────────────────────────────────────────────

async function handleDelete(id: string) {
  try {
    await deleteNotificationChannelApi(id);
    message.success('通知渠道已删除');
    await fetchChannels();
  } catch {
    message.error('删除失败');
  }
}

// ── Test ────────────────────────────────────────────────────

const testModalVisible = ref(false);
const testing = ref(false);
const testTargetId = ref('');
const testTargetName = ref('');
const testFormState = reactive({
  content: '',
  title: '测试通知',
});

function openTest(record: NotificationApi.Channel) {
  testTargetId.value = String(record.id);
  testTargetName.value = record.name;
  testFormState.title = '测试通知';
  testFormState.content = `这是一条发送到「${record.name}」的测试通知。`;
  testModalVisible.value = true;
}

async function handleConfirmTest() {
  if (!testFormState.title.trim() || !testFormState.content.trim()) {
    message.warning('请填写测试标题和内容');
    return;
  }

  testing.value = true;
  try {
    await testNotificationChannelApi({
      channelId: testTargetId.value,
      content: testFormState.content,
      title: testFormState.title,
    });
    message.success('测试通知已发送');
    testModalVisible.value = false;
  } catch {
    message.error('测试通知发送失败');
  } finally {
    testing.value = false;
  }
}

// ── Init ────────────────────────────────────────────────────

onMounted(() => {
  fetchChannels();
});
</script>

<template>
  <div class="p-5">
    <Card title="通知渠道">
      <template #extra>
        <Button type="primary" @click="openCreate">
          新增渠道
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
            <Tag color="blue">{{ channelTypeLabel(record.type) }}</Tag>
          </template>

          <template v-if="column.key === 'enabled'">
            <Tag :color="record.enabled ? 'green' : 'default'">
              {{ record.enabled ? '启用' : '停用' }}
            </Tag>
          </template>

          <template v-if="column.key === 'endpoint'">
            {{ channelEndpoint(record) }}
          </template>

          <template v-if="column.key === 'actions'">
            <Space>
              <Button
                size="small"
                type="link"
                :loading="testing && testTargetId === String(record.id)"
                @click="openTest(record as NotificationApi.Channel)"
              >
                测试
              </Button>
              <Button
                size="small"
                type="link"
                @click="openEdit(record as NotificationApi.Channel)"
              >
                编辑
              </Button>
              <Popconfirm
                title="确认删除该通知渠道？"
                @confirm="handleDelete(String(record.id))"
              >
                <Button size="small" type="link" danger>
                  删除
                </Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      v-model:open="modalVisible"
      :confirm-loading="modalLoading"
      :title="isEditing ? '编辑渠道' : '新增渠道'"
      :width="820"
      @ok="handleSubmit"
    >
      <Form
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 19 }"
        class="mt-4"
      >
        <FormItem label="名称" required>
          <Input v-model:value="formState.name" placeholder="渠道名称" />
        </FormItem>

        <FormItem label="类型" required>
          <Select
            v-model:value="formState.type"
            :options="channelTypeOptions"
            placeholder="选择渠道类型"
            @change="(value: any) => handleTypeChange(String(value))"
          />
        </FormItem>

        <FormItem label="启用">
          <Switch v-model:checked="formState.enabled" />
        </FormItem>

        <template v-if="isWebhookType">
          <FormItem label="Webhook 地址" required>
            <Input
              v-model:value="webhookForm.url"
              placeholder="https://example.com/webhook"
            />
          </FormItem>

          <FormItem label="HTTP 方法">
            <Select
              v-model:value="webhookForm.method"
              :options="methodOptions"
            />
          </FormItem>

          <FormItem label="请求头">
            <div class="channel-config-list">
              <div
                v-for="(item, index) in webhookForm.headers"
                :key="`header-${index}`"
                class="channel-config-row channel-config-row--headers"
              >
                <Input v-model:value="item.key" placeholder="Header 名称" />
                <Input v-model:value="item.value" placeholder="Header 值" />
                <Button danger @click="removeHeader(index)">
                  删除
                </Button>
              </div>
              <Button size="small" @click="addHeader">
                添加请求头
              </Button>
            </div>
          </FormItem>

          <FormItem label="请求正文" required>
            <div class="channel-config-list">
              <div
                v-for="(item, index) in webhookForm.body_fields"
                :key="`body-${index}`"
                class="channel-config-row channel-config-row--body"
              >
                <Input v-model:value="item.key" placeholder="字段名" />
                <Select
                  v-model:value="item.source"
                  :options="bodySourceOptions"
                />
                <Input
                  v-model:value="item.customValue"
                  :disabled="item.source !== 'custom'"
                  placeholder="自定义值"
                />
                <Button danger @click="removeBodyField(index)">
                  删除
                </Button>
              </div>
              <Button size="small" @click="addBodyField">
                添加字段
              </Button>
            </div>
          </FormItem>

          <FormItem label="配置预览">
            <Input.TextArea
              :value="webhookConfigPreview"
              :rows="8"
              readonly
            />
          </FormItem>
        </template>

        <FormItem v-else label="配置">
          <Input.TextArea
            v-model:value="formState.config"
            placeholder="JSON 配置"
            :rows="8"
          />
        </FormItem>

        <FormItem label="事件范围" required>
          <Select
            v-model:value="eventTags"
            mode="tags"
            :options="eventOptions"
            placeholder="输入事件类型，* 表示全部"
            :token-separators="[',', ' ']"
          />
        </FormItem>

        <FormItem label="描述">
          <Input.TextArea
            v-model:value="formState.description"
            placeholder="渠道描述"
            :rows="3"
          />
        </FormItem>
      </Form>
    </Modal>

    <Modal
      v-model:open="testModalVisible"
      :confirm-loading="testing"
      title="测试通知渠道"
      :width="560"
      @ok="handleConfirmTest"
    >
      <Form
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 19 }"
        class="mt-4"
      >
        <FormItem label="渠道">
          <Input :value="testTargetName" readonly />
        </FormItem>
        <FormItem label="标题" required>
          <Input v-model:value="testFormState.title" placeholder="测试标题" />
        </FormItem>
        <FormItem label="内容" required>
          <Input.TextArea
            v-model:value="testFormState.content"
            placeholder="测试内容"
            :rows="5"
          />
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>

<style scoped>
.channel-config-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.channel-config-row {
  display: grid;
  gap: 8px;
  width: 100%;
}

.channel-config-row--headers {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 72px;
}

.channel-config-row--body {
  grid-template-columns: minmax(0, 1.1fr) minmax(120px, 0.8fr) minmax(0, 1.1fr) 72px;
}

@media (max-width: 720px) {
  .channel-config-row,
  .channel-config-row--body,
  .channel-config-row--headers {
    grid-template-columns: 1fr;
  }
}
</style>
