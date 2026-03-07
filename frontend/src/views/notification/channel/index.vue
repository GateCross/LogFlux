<template>
  <div class="h-full">
    <n-card :title="$t('page.notification.channel.title')" :bordered="false" class="h-full rounded-2xl shadow-sm">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          {{ $t('page.notification.channel.add') }}
        </n-button>
      </template>

      <n-data-table
        remote
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        class="h-full"
        flex-height
      />
    </n-card>

    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="modalType === 'add' ? $t('page.notification.channel.add') : $t('page.notification.channel.edit')"
      class="w-760px"
    >
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="100">
        <n-form-item :label="$t('page.notification.channel.name')" path="name">
          <n-input v-model:value="formModel.name" :placeholder="$t('page.notification.channel.placeholder.name')" />
        </n-form-item>

        <n-form-item :label="$t('page.notification.channel.type')" path="type">
          <n-select
            v-model:value="formModel.type"
            :options="typeOptions"
            :placeholder="$t('page.notification.channel.placeholder.type')"
            @update:value="handleTypeChange"
          />
        </n-form-item>

        <n-form-item :label="$t('page.notification.channel.enabled')" path="enabled">
          <n-switch v-model:value="formModel.enabled" />
        </n-form-item>

        <template v-if="isWebhookType">
          <n-alert type="info" :show-icon="false" class="mb-16px">
            {{ $t('page.notification.channel.webhook.help') }}
          </n-alert>

          <n-grid :cols="2" :x-gap="12">
            <n-form-item-gi :label="$t('page.notification.channel.webhook.url')" path="config">
              <n-input v-model:value="webhookForm.url" :placeholder="$t('page.notification.channel.webhook.placeholder.url')" />
            </n-form-item-gi>

            <n-form-item-gi :label="$t('page.notification.channel.webhook.method')" path="config">
              <n-select v-model:value="webhookForm.method" :options="methodOptions" />
            </n-form-item-gi>
          </n-grid>

          <n-card size="small" class="mb-16px" :title="$t('page.notification.channel.webhook.sections.headers')">
            <n-dynamic-input v-model:value="webhookForm.headers" :on-create="createHeaderItem">
              <template #default="{ value }">
                <div class="flex w-full gap-2">
                  <n-input v-model:value="value.key" :placeholder="$t('page.notification.channel.webhook.placeholder.headerKey')" />
                  <n-input v-model:value="value.value" :placeholder="$t('page.notification.channel.webhook.placeholder.headerValue')" />
                </div>
              </template>
            </n-dynamic-input>
          </n-card>

          <n-card size="small" class="mb-16px" :title="$t('page.notification.channel.webhook.sections.body')">
            <div class="mb-12px text-sm text-gray-500">
              {{ $t('page.notification.channel.webhook.bodyHint') }}
            </div>

            <n-dynamic-input v-model:value="webhookForm.body_fields" :on-create="createBodyFieldItem">
              <template #default="{ value }">
                <div class="grid w-full grid-cols-[1.2fr_1fr_1.2fr] gap-2">
                  <n-input v-model:value="value.key" :placeholder="$t('page.notification.channel.webhook.placeholder.bodyFieldKey')" />
                  <n-select v-model:value="value.source" :options="bodySourceOptions" />
                  <n-input
                    v-model:value="value.customValue"
                    :disabled="value.source !== 'custom'"
                    :placeholder="$t('page.notification.channel.webhook.placeholder.customValue')"
                  />
                </div>
              </template>
            </n-dynamic-input>
          </n-card>

          <n-form-item :label="$t('page.notification.channel.config')">
            <n-input v-model:value="webhookConfigPreview" type="textarea" :rows="10" readonly />
          </n-form-item>
        </template>

        <n-form-item v-else :label="$t('page.notification.channel.config')" path="config">
          <n-input
            v-model:value="formModel.config"
            type="textarea"
            :placeholder="$t('page.notification.channel.placeholder.config')"
            :rows="6"
          />
        </n-form-item>

        <n-form-item :label="$t('page.notification.channel.events')" path="events">
          <n-dynamic-tags v-model:value="eventTags" />
        </n-form-item>

        <n-form-item :label="$t('page.notification.channel.description')" path="description">
          <n-input
            v-model:value="formModel.description"
            type="textarea"
            :placeholder="$t('page.notification.channel.placeholder.description')"
          />
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showModal = false">{{ $t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal
      v-model:show="showTestModal"
      preset="card"
      :title="$t('page.notification.channel.testDialog.title')"
      class="w-560px"
    >
      <n-form ref="testFormRef" :model="testFormModel" :rules="testRules" label-placement="left" label-width="90">
        <n-form-item :label="$t('page.notification.channel.testDialog.channel')">
          <n-input :value="testTargetName" readonly />
        </n-form-item>

        <n-form-item :label="$t('page.notification.channel.testDialog.titleField')" path="title">
          <n-input
            v-model:value="testFormModel.title"
            :placeholder="$t('page.notification.channel.testDialog.placeholder.title')"
          />
        </n-form-item>

        <n-form-item :label="$t('page.notification.channel.testDialog.contentField')" path="content">
          <n-input
            v-model:value="testFormModel.content"
            type="textarea"
            :rows="5"
            :placeholder="$t('page.notification.channel.testDialog.placeholder.content')"
          />
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showTestModal = false">{{ $t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="testing" @click="handleConfirmTest">{{ $t('page.notification.channel.test') }}</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { NButton, NTag, useDialog, useMessage, type DataTableColumns, type FormInst, type FormRules } from 'naive-ui';
import {
  createChannel,
  deleteChannel,
  getChannelList,
  testChannel,
  updateChannel,
  type ChannelItem,
  type TestChannelPayload,
  type WebhookBodyFieldItem,
  type WebhookConfigForm,
  type WebhookHeaderItem
} from '@/service/api/notification';

interface ChannelFormModel {
  id: number;
  name: string;
  type: string;
  enabled: boolean;
  config: string;
  events: string;
  description: string;
}

const message = useMessage();
const dialog = useDialog();
const { t } = useI18n();

const loading = ref(false);
const tableData = ref<ChannelItem[]>([]);
const pagination = ref({ page: 1, pageSize: 20 });

const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitting = ref(false);
const showTestModal = ref(false);
const testing = ref(false);
const formRef = ref<FormInst | null>(null);
const testFormRef = ref<FormInst | null>(null);

const testTargetId = ref<number>(0);
const testTargetName = ref('');

const testFormModel = ref({
  title: 'Test Notification',
  content: 'This is a test notification sent from LogFlux.'
});

const methodOptions = [
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'GET', value: 'GET' }
];

const bodySourceOptions = computed(() => [
  { label: t('page.notification.channel.webhook.bodySources.title'), value: 'title' },
  { label: t('page.notification.channel.webhook.bodySources.content'), value: 'content' },
  { label: t('page.notification.channel.webhook.bodySources.message'), value: 'message' },
  { label: t('page.notification.channel.webhook.bodySources.level'), value: 'level' },
  { label: t('page.notification.channel.webhook.bodySources.type'), value: 'type' },
  { label: t('page.notification.channel.webhook.bodySources.timestamp'), value: 'timestamp' },
  { label: t('page.notification.channel.webhook.bodySources.data'), value: 'data' },
  { label: t('page.notification.channel.webhook.bodySources.custom'), value: 'custom' }
]);

const typeOptions = [
  { label: 'Webhook', value: 'webhook' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'Slack', value: 'slack' },
  { label: '企业微信', value: 'wecom' },
  { label: '企业微信应用消息', value: 'wechat_mp' },
  { label: 'Discord', value: 'discord' },
  { label: 'Email', value: 'email' },
  { label: 'In-App', value: 'in_app' }
];

function createHeaderItem(): WebhookHeaderItem {
  return { key: '', value: '' };
}

function createBodyFieldItem(): WebhookBodyFieldItem {
  return { key: '', source: 'custom', customValue: '' };
}

function createDefaultWebhookForm(): WebhookConfigForm {
  return {
    url: '',
    method: 'POST',
    payload_mode: 'message_api',
    api_key: '',
    api_key_header: 'apiKey',
    title_field: 'title',
    content_field: 'content',
    headers: [
      { key: 'Content-Type', value: 'application/json' },
      { key: 'apiKey', value: '' }
    ],
    body_fields: [
      { key: 'title', source: 'title', customValue: '' },
      { key: 'content', source: 'content', customValue: '' }
    ]
  };
}

function createDefaultFormModel(): ChannelFormModel {
  return {
    id: 0,
    name: '',
    type: 'webhook',
    enabled: true,
    config: '',
    events: '["*"]',
    description: ''
  };
}

const formModel = ref<ChannelFormModel>(createDefaultFormModel());
const webhookForm = ref<WebhookConfigForm>(createDefaultWebhookForm());
const eventTags = ref<string[]>(['*']);

const isWebhookType = computed(() => formModel.value.type === 'webhook');
const webhookConfigPreview = computed(() => JSON.stringify(buildWebhookConfig(), null, 2));

const rules = computed<FormRules>(() => ({
  name: { required: true, message: t('form.required'), trigger: 'blur' },
  type: { required: true, message: t('form.required'), trigger: 'change' },
  config: {
    required: true,
    trigger: ['blur', 'change'],
    validator: () => {
      if (isWebhookType.value) {
        if (!webhookForm.value.url.trim()) {
          return new Error(t('page.notification.channel.webhook.validation.urlRequired'));
        }

        const hasBodyField = webhookForm.value.body_fields.some(item => item.key.trim());
        if (!hasBodyField) {
          return new Error(t('page.notification.channel.webhook.validation.bodyFieldsRequired'));
        }

        return true;
      }

      if (!formModel.value.config.trim()) {
        return new Error(t('form.required'));
      }

      try {
        JSON.parse(formModel.value.config);
        return true;
      } catch {
        return new Error(t('page.notification.channel.validation.invalidJson'));
      }
    }
  },
  events: {
    required: true,
    trigger: ['blur', 'change'],
    validator: () => {
      if (!eventTags.value.length) {
        return new Error(t('page.notification.channel.validation.eventsRequired'));
      }
      return true;
    }
  }
}));

const testRules = computed<FormRules>(() => ({
  title: { required: true, message: t('form.required'), trigger: 'blur' },
  content: { required: true, message: t('form.required'), trigger: 'blur' }
}));

const columns: DataTableColumns<ChannelItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: () => t('page.notification.channel.name'), key: 'name' },
  {
    title: () => t('page.notification.channel.type'),
    key: 'type',
    render(row) {
      return h(NTag, { type: 'info', bordered: false }, { default: () => row.type });
    }
  },
  {
    title: () => t('page.notification.channel.status'),
    key: 'enabled',
    render(row) {
      return h(
        NTag,
        { type: row.enabled ? 'success' : 'error', bordered: false },
        { default: () => (row.enabled ? t('page.notification.channel.enabled') : t('page.notification.channel.disabled')) }
      );
    }
  },
  { title: () => t('page.notification.channel.description'), key: 'description' },
  {
    title: () => t('common.action'),
    key: 'action',
    render(row) {
      return h('div', { class: 'flex gap-2' }, [
        h(NButton, { size: 'small', onClick: () => handleTest(row) }, { default: () => t('page.notification.channel.test') }),
        h(NButton, { size: 'small', onClick: () => handleEdit(row) }, { default: () => t('common.edit') }),
        h(NButton, { size: 'small', type: 'error', onClick: () => handleDelete(row) }, { default: () => t('common.delete') })
      ]);
    }
  }
];

function buildWebhookConfig() {
  const headers = webhookForm.value.headers.reduce<Record<string, string>>((acc, item) => {
    const key = item.key.trim();
    if (key) {
      acc[key] = item.value.trim();
    }
    return acc;
  }, {});

  const bodyFields = webhookForm.value.body_fields.reduce<Record<string, string>>((acc, item) => {
    const key = item.key.trim();
    if (!key) {
      return acc;
    }

    acc[key] = item.source === 'custom' ? item.customValue : item.source;
    return acc;
  }, {});

  return {
    url: webhookForm.value.url.trim(),
    method: webhookForm.value.method,
    headers,
    body_fields: bodyFields,
    payload_mode: 'message_api'
  };
}

function applyWebhookConfig(configText: string) {
  const next = createDefaultWebhookForm();
  if (!configText.trim()) {
    webhookForm.value = next;
    return;
  }

  try {
    const parsed = JSON.parse(configText);
    const headers = parsed.headers && typeof parsed.headers === 'object'
      ? Object.entries(parsed.headers as Record<string, string>).map(([key, value]) => ({ key, value: String(value ?? '') }))
      : next.headers;

    const bodyFields = parsed.body_fields && typeof parsed.body_fields === 'object'
      ? Object.entries(parsed.body_fields as Record<string, string>).map(([key, value]) => ({
          key,
          source: ['title', 'content', 'message', 'level', 'type', 'timestamp', 'data'].includes(String(value))
            ? String(value) as WebhookBodyFieldItem['source']
            : 'custom',
          customValue: ['title', 'content', 'message', 'level', 'type', 'timestamp', 'data'].includes(String(value)) ? '' : String(value ?? '')
        }))
      : next.body_fields;

    webhookForm.value = {
      ...next,
      url: typeof parsed.url === 'string' ? parsed.url : next.url,
      method: typeof parsed.method === 'string' ? parsed.method : next.method,
      headers,
      body_fields: bodyFields
    };
  } catch {
    webhookForm.value = next;
  }
}

function applyEventTags(eventsText: string) {
  if (!eventsText.trim()) {
    eventTags.value = ['*'];
    return;
  }

  try {
    const parsed = JSON.parse(eventsText);
    if (Array.isArray(parsed)) {
      eventTags.value = parsed.map(item => String(item)).filter(Boolean);
      if (!eventTags.value.length) {
        eventTags.value = ['*'];
      }
      return;
    }
  } catch {
    // ignore invalid persisted data
  }

  eventTags.value = ['*'];
}

function syncDerivedFields() {
  formModel.value.events = JSON.stringify(eventTags.value);
  if (isWebhookType.value) {
    formModel.value.config = JSON.stringify(buildWebhookConfig());
  }
}

function resetFormForAdd() {
  formModel.value = createDefaultFormModel();
  webhookForm.value = createDefaultWebhookForm();
  eventTags.value = ['*'];
  syncDerivedFields();
}

async function fetchData() {
  loading.value = true;
  try {
    const { data, error } = await getChannelList();
    if (!error && data) {
      tableData.value = data.list || [];
    }
  } finally {
    loading.value = false;
  }
}

function handleTypeChange(value: string) {
  formModel.value.type = value;
  if (value === 'webhook') {
    applyWebhookConfig(formModel.value.config);
  }
  syncDerivedFields();
}

function handleAdd() {
  modalType.value = 'add';
  resetFormForAdd();
  showModal.value = true;
}

function handleEdit(row: ChannelItem) {
  modalType.value = 'edit';
  formModel.value = { ...row };
  applyWebhookConfig(row.config || '');
  applyEventTags(row.events || '');
  syncDerivedFields();
  showModal.value = true;
}

async function handleSubmit() {
  syncDerivedFields();
  await formRef.value?.validate();
  submitting.value = true;

  try {
    const payload = {
      ...formModel.value,
      config: isWebhookType.value ? JSON.stringify(buildWebhookConfig()) : formModel.value.config,
      events: JSON.stringify(eventTags.value)
    };

    const { error } = modalType.value === 'add'
      ? await createChannel(payload)
      : await updateChannel(formModel.value.id, payload);

    if (!error) {
      message.success(t(modalType.value === 'add' ? 'common.addSuccess' : 'common.updateSuccess'));
      showModal.value = false;
      fetchData();
    } else {
      message.error(t(modalType.value === 'add' ? 'common.addFailed' : 'common.updateFailed'));
    }
  } finally {
    submitting.value = false;
  }
}

function handleDelete(row: ChannelItem) {
  dialog.warning({
    title: t('page.notification.channel.deleteConfirmTitle'),
    content: t('page.notification.channel.deleteConfirmContent', { name: row.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const { error } = await deleteChannel(row.id);
      if (!error) {
        message.success(t('common.deleteSuccess'));
        fetchData();
      } else {
        message.error(t('common.deleteFailed'));
      }
    }
  });
}

function handleTest(row: ChannelItem) {
  testTargetId.value = row.id;
  testTargetName.value = row.name;
  testFormModel.value = {
    title: 'Test Notification',
    content: `This is a test notification for channel '${row.name}'.`
  };
  showTestModal.value = true;
}

async function handleConfirmTest() {
  await testFormRef.value?.validate();
  testing.value = true;

  try {
    const payload: TestChannelPayload = {
      id: testTargetId.value,
      title: testFormModel.value.title,
      content: testFormModel.value.content
    };

    const { error } = await testChannel(payload);
    if (!error) {
      message.success(t('page.notification.channel.testSuccess'));
      showTestModal.value = false;
    } else {
      message.error(t('page.notification.channel.testFailed'));
    }
  } finally {
    testing.value = false;
  }
}

onMounted(() => {
  fetchData();
});
</script>
