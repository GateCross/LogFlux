<template>
  <div class="h-full">
    <n-card title="Notification Rules" :bordered="false" class="h-full rounded-2xl shadow-sm">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <div class="i-carbon-add" />
          </template>
          Add Rule
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

    <n-modal v-model:show="showModal" preset="card" :title="modalType === 'add' ? 'Add Rule' : 'Edit Rule'" class="w-700px">
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="120">
        <n-form-item label="Name" path="name">
          <n-input v-model:value="formModel.name" placeholder="Rule Name" />
        </n-form-item>
        
        <n-row :gutter="20">
          <n-col :span="12">
            <n-form-item label="Rule Type" path="ruleType">
              <n-select v-model:value="formModel.ruleType" :options="ruleTypeOptions" placeholder="Select Type" />
            </n-form-item>
          </n-col>
          <n-col :span="12">
            <n-form-item label="Event Type" path="eventType">
              <n-input v-model:value="formModel.eventType" placeholder="Event Type (e.g., error)" />
            </n-form-item>
          </n-col>
        </n-row>

        <n-form-item label="Enabled" path="enabled">
          <n-switch v-model:value="formModel.enabled" />
        </n-form-item>

        <n-form-item label="Condition" path="condition">
          <n-input
            v-model:value="formModel.condition"
            type="textarea"
            placeholder="JSON Condition (e.g., { 'level': 'error' })"
            :rows="3"
          />
        </n-form-item>

        <n-form-item label="Channels" path="channelIds">
          <n-select 
             v-model:value="formModel.channelIds" 
             multiple 
             :options="channelOptions" 
             placeholder="Select Channels" 
             :loading="loadingChannels"
          />
        </n-form-item>

        <n-form-item label="Template" path="template">
           <n-select 
             v-model:value="formModel.template" 
             :options="templateOptions" 
             placeholder="Select Template (Optional)" 
             clearable
             tag
             filterable
             :loading="loadingTemplates"
          />
        </n-form-item>

        <n-form-item label="Silence (sec)" path="silenceDuration">
          <n-input-number v-model:value="formModel.silenceDuration" placeholder="0" />
        </n-form-item>

        <n-form-item label="Description" path="description">
          <n-input v-model:value="formModel.description" type="textarea" placeholder="Description" />
        </n-form-item>
      </n-form>
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
import { ref, onMounted, h } from 'vue';
import { NButton, NTag, useMessage, useDialog } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { getRuleList, createRule, updateRule, deleteRule, getChannelList, getTemplateList } from '@/service/api/notification';
import type { RuleItem } from '@/service/api/notification';

const message = useMessage();
const dialog = useDialog();

const loading = ref(false);
const tableData = ref<RuleItem[]>([]);
const pagination = ref({ page: 1, pageSize: 20 });

const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitting = ref(false);
const formRef = ref();

// Options
const channelOptions = ref<{label: string, value: number}[]>([]);
const loadingChannels = ref(false);
const templateOptions = ref<{label: string, value: string}[]>([]);
const loadingTemplates = ref(false);

const ruleTypeOptions = [
  { label: 'Threshold', value: 'threshold' },
  { label: 'Frequency', value: 'frequency' },
  { label: 'Pattern', value: 'pattern' }
];

const formModel = ref({
  id: 0,
  name: '',
  enabled: true,
  ruleType: 'threshold',
  eventType: 'error',
  condition: '{}',
  channelIds: [] as number[],
  template: '',
  silenceDuration: 0,
  description: ''
});

const rules = {
  name: { required: true, message: 'Please enter name', trigger: 'blur' },
  ruleType: { required: true, message: 'Please select type', trigger: 'change' },
  eventType: { required: true, message: 'Please enter event type', trigger: 'blur' },
  channelIds: { type: 'array', required: true, message: 'Please select at least one channel', trigger: 'change' }
};

const columns: DataTableColumns<RuleItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: 'Name', key: 'name' },
  { title: 'Type', key: 'ruleType' },
  { title: 'Event', key: 'eventType' },
  { 
    title: 'Status', 
    key: 'enabled',
    render(row) {
      return h(NTag, { type: row.enabled ? 'success' : 'error', bordered: false }, { default: () => row.enabled ? 'Enabled' : 'Disabled' });
    }
  },
  { title: 'Description', key: 'description' },
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
          onClick: () => handleDelete(row)
        }, { default: () => 'Delete' })
      ]);
    }
  }
];

async function fetchData() {
  loading.value = true;
  try {
    const { data, error } = await getRuleList();
    if (!error && data) {
      tableData.value = data.list || [];
    }
  } finally {
    loading.value = false;
  }
}

async function fetchDependencies() {
  loadingChannels.value = true;
  try {
    const { data } = await getChannelList();
    if (data?.list) {
      channelOptions.value = data.list.map((c: any) => ({ label: c.name, value: c.id }));
    }
  } finally {
    loadingChannels.value = false;
  }

  loadingTemplates.value = true;
  try {
     const { data } = await getTemplateList();
     if (data?.list) {
        templateOptions.value = data.list.map((t: any) => ({ label: t.name, value: t.name }));
     }
     // Add default templates manually if not in list
     ['default_email', 'default_telegram', 'default_webhook'].forEach(t => {
        if (!templateOptions.value.find(o => o.value === t)) {
           templateOptions.value.push({ label: t + ' (System)', value: t });
        }
     });

  } finally {
    loadingTemplates.value = false;
  }
}

function handleAdd() {
  modalType.value = 'add';
  formModel.value = {
    id: 0,
    name: '',
    enabled: true,
    ruleType: 'threshold',
    eventType: 'error',
    condition: '{}',
    channelIds: [],
    template: '',
    silenceDuration: 60,
    description: ''
  };
  showModal.value = true;
}

function handleEdit(row: RuleItem) {
  modalType.value = 'edit';
  // Ensure deep copy to avoid reference issues
  formModel.value = JSON.parse(JSON.stringify(row));
  // Fix types if necessary (e.g. null to empty string)
  if(!formModel.value.condition) formModel.value.condition = '{}';
  showModal.value = true;
}

async function handleSubmit() {
  await formRef.value?.validate();
  submitting.value = true;
  try {
    const { error } = modalType.value === 'add' 
      ? await createRule(formModel.value)
      : await updateRule(formModel.value.id, formModel.value);
    
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

function handleDelete(row: RuleItem) {
  dialog.warning({
    title: 'Confirm Delete',
    content: `Are you sure to delete rule "${row.name}"?`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      const { error } = await deleteRule(row.id);
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
  fetchDependencies();
});
</script>
