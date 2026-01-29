<template>
  <div class="h-full">
    <n-card title="Notification Templates" :bordered="false" class="h-full rounded-2xl shadow-sm">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <div class="i-carbon-add" />
          </template>
          Add Template
        </n-button>
      </template>

      <n-data-table
        remote
        :columns="columns"
        :data="tableData"
        :loading="loading"
        class="h-full"
        flex-height
      />
    </n-card>

    <n-modal v-model:show="showModal" preset="card" :title="modalType === 'add' ? 'Add Template' : 'Edit Template'" class="w-1000px h-800px" :content-style="{ display: 'flex', flexDirection: 'column' }">
      <n-form ref="formRef" :model="formModel" :rules="rules" inline class="mb-4">
        <n-form-item label="Name" path="name">
          <n-input v-model:value="formModel.name" placeholder="Template Name" />
        </n-form-item>
        <n-form-item label="Format" path="format">
          <n-select v-model:value="formModel.format" :options="formatOptions" placeholder="Format" class="w-32" />
        </n-form-item>
        <n-form-item label="Type" path="type">
          <n-select v-model:value="formModel.type" :options="typeOptions" placeholder="Type" class="w-32" />
        </n-form-item>
      </n-form>
      
      <div class="flex-1 flex gap-4 min-h-0">
         <!-- Left: Editor -->
         <div class="flex-1 flex flex-col border rounded-md">
            <div class="p-2 bg-gray-50 border-b flex justify-between items-center">
               <span class="font-bold">Content Template</span>
               <n-text depth="3" class="text-xs">Supports Go Template syntax</n-text>
            </div>
            <div class="flex-1 relative">
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
         <div class="flex-1 flex flex-col border rounded-md">
            <div class="p-2 bg-gray-50 border-b flex justify-between items-center">
               <span class="font-bold">Preview</span>
               <n-button size="tiny" type="primary" @click="handlePreview" :loading="previewLoading">Refresh Preview</n-button>
            </div>
            <!-- Mock Data Input (Collapsible or small area) -->
            <div class="p-2 border-b">
               <n-input 
                 v-model:value="previewData" 
                 type="textarea" 
                 placeholder="Mock Data (JSON)" 
                 :rows="3" 
                 size="small"
               />
            </div>
            <div class="flex-1 overflow-auto p-4 bg-white">
               <!-- HTML Preview -->
               <div v-if="formModel.format === 'html'" v-html="previewContent" class="prose max-w-none"></div>
               <!-- Text/Markdown Preview -->
               <pre v-else class="whitespace-pre-wrap">{{ previewContent }}</pre>
            </div>
         </div>
      </div>
      
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showModal = false">Cancel</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">Save</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue';
import { NButton, NTag, useMessage, useDialog } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { VueMonacoEditor, loader } from '@guolao/vue-monaco-editor';
import { getTemplateList, createTemplate, updateTemplate, deleteTemplate, previewTemplate } from '@/service/api/notification';
import type { TemplateItem } from '@/service/api/notification';

// Configure Monaco Editor
loader.config({
  paths: {
    vs: 'https://registry.npmmirror.com/monaco-editor/0.44.0/files/min/vs',
  },
});

const message = useMessage();
const dialog = useDialog();

const loading = ref(false);
const tableData = ref<TemplateItem[]>([]);

const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitting = ref(false);
const formRef = ref();

const previewLoading = ref(false);
const previewContent = ref('');
const previewData = ref('{\n  "Type": "Test Event",\n  "Level": "info",\n  "Time": "2023-01-01 12:00:00",\n  "Message": "This is a test message.",\n  "Data": {"key": "value"}\n}');

const formModel = ref({
  id: 0,
  name: '',
  format: 'html',
  content: '',
  type: 'user'
});

const rules = {
  name: { required: true, message: 'Required', trigger: 'blur' },
  format: { required: true, message: 'Required', trigger: 'change' },
  content: { required: true, message: 'Required', trigger: 'blur' }
};

const formatOptions = [
  { label: 'HTML', value: 'html' },
  { label: 'Text', value: 'text' },
  { label: 'Markdown', value: 'markdown' },
  { label: 'JSON', value: 'json' }
];

const typeOptions = [
  { label: 'User', value: 'user' },
  { label: 'System', value: 'system' }
];

const editorLanguage = computed(() => {
   switch(formModel.value.format) {
      case 'html': return 'html';
      case 'json': return 'json';
      case 'markdown': return 'markdown';
      default: return 'plaintext';
   }
});

const columns: DataTableColumns<TemplateItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: 'Name', key: 'name' },
  { title: 'Format', key: 'format', width: 100 },
  { 
    title: 'Type', 
    key: 'type',
    width: 100,
    render(row) {
      return h(NTag, { type: row.type === 'system' ? 'warning' : 'default', bordered: false }, { default: () => row.type });
    }
  },
  {
    title: 'Action',
    key: 'action',
    render(row) {
      return h('div', { class: 'flex gap-2' }, [
        h(NButton, {
          size: 'small',
          onClick: () => handleEdit(row)
        }, { default: () => 'Edit' }),
        h(NButton, {
          size: 'small',
          type: 'error',
          disabled: row.type === 'system', // Protect system templates
          onClick: () => handleDelete(row)
        }, { default: () => 'Delete' })
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
    const { data } = await previewTemplate({
       format: formModel.value.format,
       content: formModel.value.content,
       data: previewData.value
    });
    if (data) {
       previewContent.value = data.content;
    }
  } catch (e) {
     previewContent.value = "Error rendering preview";
  } finally {
    previewLoading.value = false;
  }
}

async function handleSubmit() {
  await formRef.value?.validate();
  submitting.value = true;
  try {
    const { error } = modalType.value === 'add' 
      ? await createTemplate(formModel.value)
      : await updateTemplate(formModel.value.id, formModel.value);
    
    if (!error) {
      message.success('Success');
      showModal.value = false;
      fetchData();
    } else {
      message.error('Failed');
    }
  } finally {
    submitting.value = false;
  }
}

function handleDelete(row: TemplateItem) {
  dialog.warning({
    title: 'Confirm Delete',
    content: `Are you sure to delete template "${row.name}"?`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      const { error } = await deleteTemplate(row.id);
      if (!error) {
        message.success('Deleted');
        fetchData();
      } else {
        message.error('Delete failed');
      }
    }
  });
}

onMounted(() => {
  fetchData();
});
</script>
