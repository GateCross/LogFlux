<template>
  <div class="h-full">
    <n-card title="Notification Channels" :bordered="false" class="h-full rounded-2xl shadow-sm">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <div class="i-carbon-add" />
          </template>
          Add Channel
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

    <n-modal v-model:show="showModal" preset="card" :title="modalType === 'add' ? 'Add Channel' : 'Edit Channel'" class="w-600px">
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="100">
        <n-form-item label="Name" path="name">
          <n-input v-model:value="formModel.name" placeholder="Channel Name" />
        </n-form-item>
        <n-form-item label="Type" path="type">
          <n-select v-model:value="formModel.type" :options="typeOptions" placeholder="Select Type" />
        </n-form-item>
        <n-form-item label="Enabled" path="enabled">
          <n-switch v-model:value="formModel.enabled" />
        </n-form-item>
        <n-form-item label="Config" path="config">
          <n-input
            v-model:value="formModel.config"
            type="textarea"
            placeholder="JSON Configuration (e.g., { 'webhook_url': '...' })"
            :rows="5"
          />
        </n-form-item>
        <n-form-item label="Events" path="events">
           <n-input v-model:value="formModel.events" placeholder='["*"] or ["error", "caddy"]' />
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
import { getChannelList, createChannel, updateChannel, deleteChannel, testChannel } from '@/service/api/notification';
import type { ChannelItem } from '@/service/api/notification';

const message = useMessage();
const dialog = useDialog();

const loading = ref(false);
const tableData = ref<ChannelItem[]>([]);
const pagination = ref({ page: 1, pageSize: 20 });

const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitting = ref(false);
const formRef = ref();

const formModel = ref({
  id: 0,
  name: '',
  type: 'webhook',
  enabled: true,
  config: '{}',
  events: '["*"]',
  description: ''
});

const rules = {
  name: { required: true, message: 'Please enter name', trigger: 'blur' },
  type: { required: true, message: 'Please select type', trigger: 'change' },
  config: { required: true, message: 'Please enter config', trigger: 'blur' }
};

const typeOptions = [
  { label: 'Webhook', value: 'webhook' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'Email', value: 'email' }
];

const columns: DataTableColumns<ChannelItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: 'Name', key: 'name' },
  { 
    title: 'Type', 
    key: 'type',
    render(row) {
      return h(NTag, { type: 'info', bordered: false }, { default: () => row.type });
    }
  },
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
          onClick: () => handleTest(row)
        }, { default: () => 'Test' }),
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
    const { data, error } = await getChannelList();
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
    type: 'webhook',
    enabled: true,
    config: '{\n  "url": ""\n}',
    events: '["*"]',
    description: ''
  };
  showModal.value = true;
}

function handleEdit(row: ChannelItem) {
  modalType.value = 'edit';
  formModel.value = { ...row };
  showModal.value = true;
}

async function handleSubmit() {
  await formRef.value?.validate();
  submitting.value = true;
  try {
    const { error } = modalType.value === 'add' 
      ? await createChannel(formModel.value)
      : await updateChannel(formModel.value.id, formModel.value);
    
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

function handleDelete(row: ChannelItem) {
  dialog.warning({
    title: 'Confirm Delete',
    content: `Are you sure to delete channel "${row.name}"?`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      const { error } = await deleteChannel(row.id);
      if (!error) {
        message.success('Deleted');
        fetchData();
      } else {
        message.error('Delete failed');
      }
    }
  });
}

async function handleTest(row: ChannelItem) {
  const { error } = await testChannel(row.id);
  if (!error) {
    message.success('Test notification sent');
  } else {
    message.error('Test failed');
  }
}

onMounted(() => {
  fetchData();
});
</script>
