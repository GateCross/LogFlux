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

    <n-modal v-model:show="showModal" preset="card" :title="modalType === 'add' ? $t('page.notification.channel.add') : $t('page.notification.channel.edit')" class="w-600px">
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="100">
        <n-form-item :label="$t('page.notification.channel.name')" path="name">
          <n-input v-model:value="formModel.name" :placeholder="$t('page.notification.channel.placeholder.name')" />
        </n-form-item>
        <n-form-item :label="$t('page.notification.channel.type')" path="type">
          <n-select v-model:value="formModel.type" :options="typeOptions" :placeholder="$t('page.notification.channel.placeholder.type')" />
        </n-form-item>
        <n-form-item :label="$t('page.notification.channel.enabled')" path="enabled">
          <n-switch v-model:value="formModel.enabled" />
        </n-form-item>
        <n-form-item :label="$t('page.notification.channel.config')" path="config">
          <n-input
            v-model:value="formModel.config"
            type="textarea"
            :placeholder="$t('page.notification.channel.placeholder.config')"
            :rows="5"
          />
        </n-form-item>
        <n-form-item :label="$t('page.notification.channel.events')" path="events">
           <n-input v-model:value="formModel.events" :placeholder="$t('page.notification.channel.placeholder.events')" />
        </n-form-item>
        <n-form-item :label="$t('page.notification.channel.description')" path="description">
          <n-input v-model:value="formModel.description" type="textarea" :placeholder="$t('page.notification.channel.placeholder.description')" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showModal = false">{{ $t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { NButton, NTag, useMessage, useDialog } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { getChannelList, createChannel, updateChannel, deleteChannel, testChannel } from '@/service/api/notification';
import type { ChannelItem } from '@/service/api/notification';

const message = useMessage();
const dialog = useDialog();
const { t } = useI18n();

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

const rules = computed(() => ({
  name: { required: true, message: t('form.required'), trigger: 'blur' },
  type: { required: true, message: t('form.required'), trigger: 'change' },
  config: { required: true, message: t('form.required'), trigger: 'blur' }
}));

const typeOptions = [
  { label: 'Webhook', value: 'webhook' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'Slack', value: 'slack' },
  { label: 'Discord', value: 'discord' },
  { label: 'Email', value: 'email' },
  { label: 'In-App', value: 'in_app' }
];

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
      return h(NTag, { type: row.enabled ? 'success' : 'error', bordered: false }, { default: () => row.enabled ? t('page.notification.channel.enabled') : t('page.notification.channel.disabled') });
    }
  },
  { title: () => t('page.notification.channel.description'), key: 'description' },
  {
    title: () => t('common.action'),
    key: 'action',
    render(row) {
      return h('div', { class: 'flex gap-2' }, [
        h(NButton, {
          size: 'small',
          onClick: () => handleTest(row)
        }, { default: () => t('page.notification.channel.test') }),
        h(NButton, {
          size: 'small',
          onClick: () => handleEdit(row)
        }, { default: () => t('common.edit') }),
        h(NButton, {
          size: 'small',
          type: 'error',
          onClick: () => handleDelete(row)
        }, { default: () => t('common.delete') })
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

async function handleTest(row: ChannelItem) {
  const { error } = await testChannel(row.id);
  if (!error) {
    message.success(t('page.notification.channel.testSuccess'));
  } else {
    message.error(t('page.notification.channel.testFailed'));
  }
}

onMounted(() => {
  fetchData();
});
</script>
