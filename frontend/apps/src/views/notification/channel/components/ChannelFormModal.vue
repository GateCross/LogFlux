<script lang="ts" setup>
import { computed, reactive, ref, watch } from 'vue';

import {
  Button,
  Form,
  FormItem,
  Input,
  Modal,
  Select,
  Switch,
  message,
} from 'antdv-next';

import type { NotificationApi } from '#/api/notification';
import {
  createNotificationChannelApi,
  updateNotificationChannelApi,
} from '#/api/notification';

import {
  type ChannelFormState,
  type WebhookConfigForm,
  applyEventTags,
  applyWebhookConfig,
  bodySourceOptions,
  buildWebhookConfig,
  channelTypeOptions,
  createBodyFieldItem,
  createDefaultFormState,
  createDefaultWebhookForm,
  createHeaderItem,
  eventOptions,
  methodOptions,
  parseConfig,
  stringifyConfig,
} from '../channel-utils';

const props = defineProps<{
  open: boolean;
  /** 编辑时传入渠道记录；新增时为 null */
  channel: NotificationApi.Channel | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  success: [];
}>();

const modalLoading = ref(false);
const formState = reactive<ChannelFormState>(createDefaultFormState());
const webhookForm = ref<WebhookConfigForm>(createDefaultWebhookForm());
const eventTags = ref<string[]>(['*']);

const isEditing = computed(() => props.channel !== null);
const isWebhookType = computed(() => formState.type === 'webhook');
const webhookConfigPreview = computed(() =>
  JSON.stringify(buildWebhookConfig(webhookForm.value), null, 2),
);

function resetFormForCreate() {
  Object.assign(formState, createDefaultFormState());
  webhookForm.value = createDefaultWebhookForm();
  eventTags.value = ['*'];
}

function applyChannel(record: NotificationApi.Channel) {
  Object.assign(formState, {
    config: stringifyConfig(record.config),
    description: record.description ?? '',
    enabled: record.enabled ?? record.status !== 'disabled',
    events: record.events ?? '["*"]',
    name: record.name,
    type: record.type,
  });
  eventTags.value = applyEventTags(formState.events);
  webhookForm.value = applyWebhookConfig(record.config);
}

watch(() => props.open,
  (open) => {
    if (!open) return;
    if (props.channel) {
      applyChannel(props.channel);
    } else {
      resetFormForCreate();
    }
  },
);

function handleTypeChange(value: string) {
  formState.type = value;
  if (value === 'webhook') {
    webhookForm.value = applyWebhookConfig(formState.config);
  }
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
      ? buildWebhookConfig(webhookForm.value)
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

    if (isEditing.value && props.channel) {
      await updateNotificationChannelApi(String(props.channel.id), payload);
      message.success('通知渠道已更新');
    } else {
      await createNotificationChannelApi(payload);
      message.success('通知渠道已创建');
    }
    emit('update:open', false);
    emit('success');
  } catch {
    message.error('操作失败');
  } finally {
    modalLoading.value = false;
  }
}

function handleOpenChange(value: boolean) {
  emit('update:open', value);
}
</script>

<template>
  <Modal
    :open="open"
    :confirm-loading="modalLoading"
    :title="isEditing ? '编辑渠道' : '新增渠道'"
    :width="820"
    @update:open="handleOpenChange"
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
