<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import { NButton, NTag, useDialog, useMessage } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { useI18n } from 'vue-i18n';
import { VueMonacoEditor, loader } from '@guolao/vue-monaco-editor';
import {
  createTemplate,
  deleteTemplate,
  getTemplateList,
  previewTemplate,
  updateTemplate
} from '@/service/api/notification';
import type { TemplateItem } from '@/service/api/notification';

// Configure Monaco Editor
loader.config({
  paths: {
    vs: 'https://registry.npmmirror.com/monaco-editor/0.44.0/files/min/vs'
  }
});

const message = useMessage();
const dialog = useDialog();
const { t } = useI18n();

const loading = ref(false);
const tableData = ref<TemplateItem[]>([]);

const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitting = ref(false);
const formRef = ref();

const previewLoading = ref(false);
const previewContent = ref('');
const previewData = ref(
  '{\n  "Type": "Test Event",\n  "Level": "info",\n  "Time": "2023-01-01 12:00:00",\n  "Message": "This is a test message.",\n  "Data": {"key": "value"}\n}'
);

const formModel = ref({
  id: 0,
  name: '',
  format: 'html',
  content: '',
  type: 'user'
});

const rules = computed(() => ({
  name: { required: true, message: t('form.required'), trigger: 'blur' },
  format: { required: true, message: t('form.required'), trigger: 'change' },
  content: { required: true, message: t('form.required'), trigger: 'blur' }
}));

const formatOptions = computed(() => [
  { label: t('page.notification.template.formats.html'), value: 'html' },
  { label: t('page.notification.template.formats.text'), value: 'text' },
  { label: t('page.notification.template.formats.markdown'), value: 'markdown' },
  { label: t('page.notification.template.formats.json'), value: 'json' }
]);

const typeOptions = computed(() => [
  { label: t('page.notification.template.types.user'), value: 'user' },
  { label: t('page.notification.template.types.system'), value: 'system' }
]);

const editorLanguage = computed(() => {
  switch (formModel.value.format) {
    case 'html':
      return 'html';
    case 'json':
      return 'json';
    case 'markdown':
      return 'markdown';
    default:
      return 'plaintext';
  }
});

const columns: DataTableColumns<TemplateItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: () => t('page.notification.template.name'), key: 'name' },
  { title: () => t('page.notification.template.format'), key: 'format', width: 100 },
  {
    title: () => t('page.notification.template.type'),
    key: 'type',
    width: 100,
    render(row) {
      return h(
        NTag,
        { type: row.type === 'system' ? 'warning' : 'default', bordered: false },
        { default: () => row.type }
      );
    }
  },
  {
    title: () => t('common.action'),
    key: 'action',
    render(row) {
      return h('div', { class: 'flex gap-2' }, [
        h(
          NButton,
          {
            size: 'small',
            onClick: () => handleEdit(row)
          },
          { default: () => t('common.edit') }
        ),
        h(
          NButton,
          {
            size: 'small',
            type: 'error',
            disabled: row.type === 'system', // Protect system templates
            onClick: () => handleDelete(row)
          },
          { default: () => t('common.delete') }
        )
      ]);
    }
  }
];

async function fetchData() {
  loading.value = true;
  try {
    const { data, error } = await getTemplateList();
    if (!error && data) {
      tableData.value = data.list || [];
    }
  } finally {
    loading.value = false;
  }
}

function handleAdd() {
  modalType.value = 'add';
  formModel.value = {
    id: 0,
    name: '',
    format: 'html',
    content: '<div>{{.Message}}</div>',
    type: 'user'
  };
  previewContent.value = '';
  showModal.value = true;
}

function handleEdit(row: TemplateItem) {
  modalType.value = 'edit';
  formModel.value = { ...row };
  previewContent.value = '';
  showModal.value = true;
  handlePreview(); // Initial preview
}

async function handlePreview() {
  previewLoading.value = true;
  try {
    let parsedData: Record<string, any> | undefined;
    try {
      parsedData = JSON.parse(previewData.value);
    } catch {
      parsedData = undefined;
    }
    const { data } = await previewTemplate({
      format: formModel.value.format,
      content: formModel.value.content,
      data: parsedData
    });
    if (data) {
      previewContent.value = data.content;
    }
  } catch (e) {
    previewContent.value = 'Error rendering preview';
  } finally {
    previewLoading.value = false;
  }
}

async function handleSubmit() {
  await formRef.value?.validate();
  submitting.value = true;
  try {
    const { error } =
      modalType.value === 'add'
        ? await createTemplate(formModel.value)
        : await updateTemplate(formModel.value.id, formModel.value);

    if (!error) {
      message.success(t('common.addSuccess'));
      showModal.value = false;
      fetchData();
    } else {
      message.error(t('common.updateFailed'));
    }
  } finally {
    submitting.value = false;
  }
}

function handleDelete(row: TemplateItem) {
  dialog.warning({
    title: t('page.notification.template.deleteConfirmTitle'),
    content: t('page.notification.template.deleteConfirmContent', { name: row.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const { error } = await deleteTemplate(row.id);
      if (!error) {
        message.success(t('common.deleteSuccess'));
        fetchData();
      } else {
        message.error(t('common.deleteFailed'));
      }
    }
  });
}

onMounted(() => {
  fetchData();
});
</script>

<template>
  <div class="h-full">
    <NCard :title="$t('page.notification.template.title')" :bordered="false" class="h-full rounded-2xl shadow-sm">
      <template #header-extra>
        <NButton type="primary" @click="handleAdd">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          {{ $t('page.notification.template.add') }}
        </NButton>
      </template>

      <NDataTable remote :columns="columns" :data="tableData" :loading="loading" class="h-full" flex-height />
    </NCard>

    <NModal
      v-model:show="showModal"
      preset="card"
      :title="modalType === 'add' ? $t('page.notification.template.add') : $t('page.notification.template.edit')"
      class="h-800px w-1000px"
      :content-style="{ display: 'flex', flexDirection: 'column' }"
    >
      <NForm ref="formRef" :model="formModel" :rules="rules" inline class="mb-4">
        <NFormItem :label="$t('page.notification.template.name')" path="name">
          <NInput v-model:value="formModel.name" :placeholder="$t('page.notification.template.placeholder.name')" />
        </NFormItem>
        <NFormItem :label="$t('page.notification.template.format')" path="format">
          <NSelect
            v-model:value="formModel.format"
            :options="formatOptions"
            :placeholder="$t('page.notification.template.placeholder.format')"
            class="w-32"
          />
        </NFormItem>
        <NFormItem :label="$t('page.notification.template.type')" path="type">
          <NSelect
            v-model:value="formModel.type"
            :options="typeOptions"
            :placeholder="$t('page.notification.template.placeholder.type')"
            class="w-32"
          />
        </NFormItem>
      </NForm>

      <div class="min-h-0 flex flex-1 gap-4">
        <!-- Left: Editor -->
        <div class="flex flex-col flex-1 border rounded-md">
          <div class="flex items-center justify-between border-b bg-gray-50 p-2">
            <span class="font-bold">{{ $t('page.notification.template.content') }}</span>
            <NText depth="3" class="text-xs">Supports Go Template syntax</NText>
          </div>
          <div class="relative flex-1">
            <VueMonacoEditor
              v-model:value="formModel.content"
              :language="editorLanguage"
              theme="vs"
              :options="{
                automaticLayout: true,
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                wordWrap: 'on'
              }"
              class="absolute inset-0"
            />
          </div>
        </div>

        <!-- Right: Preview -->
        <div class="flex flex-col flex-1 border rounded-md">
          <div class="flex items-center justify-between border-b bg-gray-50 p-2">
            <span class="font-bold">{{ $t('page.notification.template.preview') }}</span>
            <NButton size="tiny" type="primary" :loading="previewLoading" @click="handlePreview">
              {{ $t('page.notification.template.refreshPreview') }}
            </NButton>
          </div>
          <!-- Mock Data Input (Collapsible or small area) -->
          <div class="border-b p-2">
            <NInput
              v-model:value="previewData"
              type="textarea"
              :placeholder="$t('page.notification.template.placeholder.mockData')"
              :rows="3"
              size="small"
            />
          </div>
          <div class="flex-1 overflow-auto bg-white p-4">
            <!-- HTML Preview -->
            <div v-if="formModel.format === 'html'" class="prose max-w-none" v-html="previewContent"></div>
            <!-- Text/Markdown Preview -->
            <pre v-else class="whitespace-pre-wrap">{{ previewContent }}</pre>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="showModal = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>
