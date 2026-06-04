<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import { computed, onMounted, reactive, ref } from 'vue';

import {
  Button,
  Card,
  Drawer,
  Form,
  FormItem,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  message,
} from 'ant-design-vue';

import {
  createNotificationTemplateApi,
  deleteNotificationTemplateApi,
  getNotificationTemplatesApi,
  previewNotificationTemplateApi,
  updateNotificationTemplateApi,
} from '#/api/notification';

defineOptions({ name: 'NotificationTemplate' });

// ── List state ──────────────────────────────────────────────

const loading = ref(false);
const templates = ref<NotificationApi.Template[]>([]);

async function fetchTemplates() {
  loading.value = true;
  try {
    templates.value = await getNotificationTemplatesApi();
  } finally {
    loading.value = false;
  }
}

// ── Table columns ───────────────────────────────────────────

const columns = [
  { dataIndex: 'name', key: 'name', title: '名称' },
  { dataIndex: 'type', key: 'type', title: '类型' },
  { dataIndex: 'subject', key: 'subject', title: '主题', ellipsis: true },
  { key: 'actions', title: '操作', width: 240 },
];

// ── Template type options ───────────────────────────────────

const templateTypeOptions = [
  { label: 'Email', value: 'email' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'DingTalk', value: 'dingtalk' },
  { label: 'WeChat', value: 'wechat' },
  { label: 'Slack', value: 'slack' },
  { label: 'Webhook', value: 'webhook' },
];

// ── 创建 / 编辑 modal ─────────────────────────────────────

const modalVisible = ref(false);
const modalLoading = ref(false);
const editingId = ref<string | null>(null);
const is编辑ing = computed(() => editingId.value !== null);

const formState = reactive<NotificationApi.TemplateParams>({
  content: '',
  name: '',
  type: 'email',
});

function openCreate() {
  editingId.value = null;
  Object.assign(formState, {
    content: '',
    name: '',
    type: 'email',
  });
  modalVisible.value = true;
}

function open编辑(record: NotificationApi.Template) {
  editingId.value = record.id;
  Object.assign(formState, {
    content: record.content,
    name: record.name,
    type: record.type,
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  modalLoading.value = true;
  try {
    if (is编辑ing.value) {
      await updateNotificationTemplateApi(editingId.value!, { ...formState });
      message.success('Template updated 成功');
    } else {
      await createNotificationTemplateApi({ ...formState });
      message.success('Template created 成功');
    }
    modalVisible.value = false;
    await fetchTemplates();
  } catch {
    message.error('操作失败');
  } finally {
    modalLoading.value = false;
  }
}

// ── 删除 ──────────────────────────────────────────────────

async function handle删除(id: string) {
  try {
    await deleteNotificationTemplateApi(id);
    message.success('Template deleted');
    await fetchTemplates();
  } catch {
    message.error('删除 failed');
  }
}

// ── Preview drawer ──────────────────────────────────────────

const previewDrawerVisible = ref(false);
const previewLoading = ref(false);
const previewContent = ref('');
const previewTemplate = ref<NotificationApi.Template | null>(null);

const previewVariables = ref('{}');

async function openPreview(record: NotificationApi.Template) {
  previewTemplate.value = record;
  previewContent.value = '';
  previewVariables.value = '{}';
  previewDrawerVisible.value = true;
  await handlePreview();
}

async function handlePreview() {
  if (!previewTemplate.value) return;
  previewLoading.value = true;
  try {
    let variables: Record<string, any> = {};
    try {
      variables = JSON.parse(previewVariables.value);
    } catch {
      message.warning('Invalid JSON in variables field');
    }
    const result = await previewNotificationTemplateApi({
      content: previewTemplate.value.content,
      variables,
    });
    previewContent.value = result.rendered;
  } catch {
    message.error('Preview failed');
  } finally {
    previewLoading.value = false;
  }
}

// ── Init ────────────────────────────────────────────────────

onMounted(() => {
  fetchTemplates();
});
</script>

<template>
  <div class="p-5">
    <Card title="Notification Templates">
      <template #extra>
        <Button type="primary" @click="openCreate">
          新增 Template
        </Button>
      </template>

      <Table
        :columns="columns"
        :data-source="templates"
        :loading="loading"
        row-key="id"
        size="middle"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <Tag color="blue">{{ record.type }}</Tag>
          </template>

          <template v-if="column.key === 'subject'">
            {{ record.content?.substring(0, 60) }}{{ record.content?.length > 60 ? '...' : '' }}
          </template>

          <template v-if="column.key === 'actions'">
            <Space>
              <Button
                size="small"
                type="link"
                @click="openPreview(record as NotificationApi.Template)"
              >
                Preview
              </Button>
              <Button
                size="small"
                type="link"
                @click="open编辑(record as NotificationApi.Template)"
              >
                编辑
              </Button>
              <Popconfirm
                title="Are you sure to delete this template?"
                @confirm="handle删除(record.id)"
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

    <!-- ── 创建 / 编辑 modal ──────────────────────────────── -->
    <Modal
      v-model:open="modalVisible"
      :confirm-loading="modalLoading"
      :title="is编辑ing ? '编辑 Template' : '新增 Template'"
      :width="640"
      @ok="handleSubmit"
    >
      <Form
        :label-col="{ span: 4 }"
        :wrapper-col="{ span: 20 }"
        class="mt-4"
      >
        <FormItem label="Name" required>
          <Input v-model:value="formState.name" placeholder="Template name" />
        </FormItem>
        <FormItem label="Type" required>
          <Select
            v-model:value="formState.type"
            :options="templateTypeOptions"
            placeholder="Select template type"
          />
        </FormItem>
        <FormItem label="Content" required>
          <Input.TextArea
            v-model:value="formState.content"
            :rows="10"
            placeholder="Template content (supports variables like {{.Name}})"
          />
        </FormItem>
      </Form>
    </Modal>

    <!-- ── Preview drawer ───────────────────────────────────── -->
    <Drawer
      v-model:open="previewDrawerVisible"
      title="Template Preview"
      :width="640"
    >
      <Form :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <FormItem label="Variables (JSON)">
          <Input.TextArea
            v-model:value="previewVariables"
            :rows="4"
            placeholder='{"key": "value"}'
          />
        </FormItem>
        <FormItem :wrapper-col="{ offset: 6, span: 18 }">
          <Button type="primary" :loading="previewLoading" @click="handlePreview">
            Refresh Preview
          </Button>
        </FormItem>
      </Form>

      <div style="margin-top: 16px;">
        <h4>Rendered Result:</h4>
        <div
          style="
            background: #f5f5f5;
            border: 1px solid #e8e8e8;
            border-radius: 4px;
            padding: 16px;
            white-space: pre-wrap;
            word-break: break-all;
            min-height: 120px;
            font-family: monospace;
            font-size: 13px;
          "
        >
          {{ previewContent || 'No preview available' }}
        </div>
      </div>
    </Drawer>
  </div>
</template>
