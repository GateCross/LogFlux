<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import { computed, reactive, ref } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import {
  Alert,
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
} from 'antdv-next';

import {
  createNotificationTemplateApi,
  deleteNotificationTemplateApi,
  getNotificationTemplatesApi,
  previewNotificationTemplateApi,
  updateNotificationTemplateApi,
} from '#/api/notification';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

defineOptions({ name: 'NotificationTemplate' });

const queryClient = useQueryClient();

const {
  data: templatesData,
  loading,
  errorMessage,
  refetch,
} = useListDetailQuery({
  queryKey: qk.notification.templates(),
  queryFn: () => getNotificationTemplatesApi(withListDetailErrorMode()),
  errorFallback: '加载通知模板失败',
});

const templates = computed(() => templatesData.value ?? []);

const columns = [
  { dataIndex: 'name', key: 'name', title: '名称' },
  { dataIndex: 'type', key: 'type', title: '类型' },
  { dataIndex: 'subject', key: 'subject', title: '主题', ellipsis: true },
  { key: 'actions', title: '操作', width: 280 },
];

const templateFormatOptions = [
  { label: '文本 (Text)', value: 'text' },
  { label: '富文本 (HTML)', value: 'html' },
  { label: 'Markdown', value: 'markdown' },
  { label: 'JSON', value: 'json' },
];

const templateTypeLabels: Record<string, string> = {
  system: '系统',
  user: '自定义',
};

const modalVisible = ref(false);
const modalLoading = ref(false);
const editingId = ref<string | null>(null);
const isEditing = computed(() => editingId.value !== null);

const formState = reactive<NotificationApi.TemplateParams>({
  content: '',
  name: '',
  format: 'text',
  type: 'user',
});

function openCreate() {
  editingId.value = null;
  Object.assign(formState, {
    content: '',
    name: '',
    format: 'text',
    type: 'user',
  });
  modalVisible.value = true;
}

function openEdit(record: NotificationApi.Template) {
  editingId.value = record.id;
  Object.assign(formState, {
    content: record.content,
    name: record.name,
    format: record.format || 'text',
    type: record.type,
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  modalLoading.value = true;
  try {
    if (isEditing.value) {
      await updateNotificationTemplateApi(editingId.value!, { ...formState });
      message.success('通知模板已更新');
    } else {
      await createNotificationTemplateApi({ ...formState });
      message.success('通知模板已创建');
    }
    modalVisible.value = false;
    await invalidateListDetailQueries(queryClient, qk.notification.templates());
    await refetch();
  } catch {
    message.error('操作失败');
  } finally {
    modalLoading.value = false;
  }
}

async function handleDelete(id: string) {
  try {
    await deleteNotificationTemplateApi(id);
    message.success('通知模板已删除');
    await invalidateListDetailQueries(queryClient, qk.notification.templates());
    await refetch();
  } catch {
    message.error('删除失败');
  }
}

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
      message.warning('变量字段不是有效 JSON');
    }
    const result = await previewNotificationTemplateApi({
      content: previewTemplate.value.content,
      variables,
    });
    previewContent.value = result.rendered;
  } catch {
    message.error('预览失败');
  } finally {
    previewLoading.value = false;
  }
}

function templateFormatLabel(format: string) {
  return templateFormatOptions.find((item) => item.value === format)?.label ?? format;
}
</script>

<template>
  <div class="p-5">
    <Alert
      v-if="errorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="errorMessage"
    />
    <Card title="通知模板">
      <template #extra>
        <Button type="primary" @click="openCreate">新增模板</Button>
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
            <Space>
              <Tag :color="record.type === 'system' ? 'purple' : 'blue'">
                {{ templateTypeLabels[record.type] ?? record.type }}
              </Tag>
              <Tag color="cyan">{{ templateFormatLabel(record.format || 'text') }}</Tag>
            </Space>
          </template>

          <template v-if="column.key === 'subject'">
            {{ record.content?.substring(0, 60)
            }}{{ record.content?.length > 60 ? '...' : '' }}
          </template>

          <template v-if="column.key === 'actions'">
            <Space :size="6">
              <Button
                size="small"
                class="table-action-btn table-action-btn--secondary"
                @click="openPreview(record as NotificationApi.Template)"
              >
                预览
              </Button>
              <Button
                size="small"
                class="table-action-btn"
                @click="openEdit(record as NotificationApi.Template)"
              >
                编辑
              </Button>
              <Popconfirm title="确认删除该通知模板？" @confirm="handleDelete(record.id)">
                <Button size="small" danger class="table-action-btn table-action-btn--danger">
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
      :title="isEditing ? '编辑模板' : '新增模板'"
      :width="640"
      @ok="handleSubmit"
    >
      <Form :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }" class="mt-4">
        <FormItem label="名称" required>
          <Input
            v-model:value="formState.name"
            placeholder="模板名称"
            :disabled="formState.type === 'system'"
          />
        </FormItem>
        <FormItem label="格式" required>
          <Select
            v-model:value="formState.format"
            :options="templateFormatOptions"
            placeholder="选择模板格式"
          />
        </FormItem>
        <FormItem v-if="isEditing" label="类型">
          <Tag :color="formState.type === 'system' ? 'purple' : 'blue'">
            {{ templateTypeLabels[formState.type] ?? formState.type }}
          </Tag>
          <span v-if="formState.type === 'system'" class="text-gray-400 text-xs ml-2">
            系统模板仅可修改格式和内容
          </span>
        </FormItem>
        <FormItem label="内容" required>
          <Input.TextArea
            v-model:value="formState.content"
            :rows="10"
            placeholder="模板内容，支持 {{.Name}} 这样的变量"
          />
        </FormItem>
      </Form>
    </Modal>

    <Drawer v-model:open="previewDrawerVisible" title="模板预览" :size="640">
      <Form :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <FormItem label="变量 JSON">
          <Input.TextArea
            v-model:value="previewVariables"
            :rows="4"
            placeholder='{"key": "value"}'
          />
        </FormItem>
        <FormItem :wrapper-col="{ offset: 6, span: 18 }">
          <Button type="primary" :loading="previewLoading" @click="handlePreview">
            刷新预览
          </Button>
        </FormItem>
      </Form>

      <div style="margin-top: 16px">
        <h4>渲染结果：</h4>
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
          {{ previewContent || '暂无预览内容' }}
        </div>
      </div>
    </Drawer>
  </div>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
