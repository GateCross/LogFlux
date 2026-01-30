<template>
  <div class="h-full">
    <n-card :title="$t('page.notification.rule.title')" :bordered="false" class="h-full rounded-2xl shadow-sm">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          {{ $t('page.notification.rule.add') }}
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

    <n-modal v-model:show="showModal" preset="card" :title="modalType === 'add' ? $t('page.notification.rule.add') : $t('page.notification.rule.edit')" class="w-700px">
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="120">
        <n-form-item :label="$t('page.notification.rule.name')" path="name">
          <n-input v-model:value="formModel.name" :placeholder="$t('page.notification.rule.placeholder.name')" />
        </n-form-item>
        
        <n-row :gutter="20">
          <n-col :span="12">
            <n-form-item :label="$t('page.notification.rule.ruleType')" path="ruleType">
              <n-select v-model:value="formModel.ruleType" :options="ruleTypeOptions" :placeholder="$t('page.notification.rule.placeholder.type')" />
            </n-form-item>
          </n-col>
          <n-col :span="12">
            <n-form-item :label="$t('page.notification.rule.eventType')" path="eventType">
              <n-input v-model:value="formModel.eventType" :placeholder="$t('page.notification.rule.placeholder.eventType')" />
            </n-form-item>
          </n-col>
        </n-row>

        <n-form-item :label="$t('page.notification.rule.enabled')" path="enabled">
          <n-switch v-model:value="formModel.enabled" />
        </n-form-item>

        <n-form-item :label="$t('page.notification.rule.condition')" path="condition">
          <n-input
            v-model:value="formModel.condition"
            type="textarea"
            :placeholder="$t('page.notification.rule.placeholder.condition')"
            :rows="3"
          />
        </n-form-item>

        <n-form-item :label="$t('page.notification.rule.channels')" path="channelIds">
          <n-select 
             v-model:value="formModel.channelIds" 
             multiple 
             :options="channelOptions" 
             :placeholder="$t('page.notification.rule.placeholder.channels')" 
             :loading="loadingChannels"
          />
        </n-form-item>

        <n-form-item :label="$t('page.notification.rule.template')" path="template">
           <n-select 
             v-model:value="formModel.template" 
             :options="templateOptions" 
             :placeholder="$t('page.notification.rule.placeholder.template')" 
             clearable
             tag
             filterable
             :loading="loadingTemplates"
          />
        </n-form-item>

        <n-form-item :label="$t('page.notification.rule.silence')" path="silenceDuration">
          <n-input-number v-model:value="formModel.silenceDuration" placeholder="0" />
        </n-form-item>

        <n-form-item :label="$t('page.notification.rule.description')" path="description">
          <n-input v-model:value="formModel.description" type="textarea" :placeholder="$t('page.notification.rule.placeholder.description')" />
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
import { getRuleList, createRule, updateRule, deleteRule, getChannelList, getTemplateList } from '@/service/api/notification';
import type { RuleItem } from '@/service/api/notification';

const message = useMessage();
const dialog = useDialog();
const { t } = useI18n();

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

const ruleTypeOptions = computed(() => [
  { label: t('page.notification.rule.types.threshold'), value: 'threshold' },
  { label: t('page.notification.rule.types.frequency'), value: 'frequency' },
  { label: t('page.notification.rule.types.pattern'), value: 'pattern' }
]);

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

const rules = computed(() => ({
  name: { required: true, message: t('form.required'), trigger: 'blur' },
  ruleType: { required: true, message: t('form.required'), trigger: 'change' },
  eventType: { required: true, message: t('form.required'), trigger: 'blur' },
  channelIds: { type: 'array', required: true, message: t('form.required'), trigger: 'change' } as unknown as import('naive-ui').FormItemRule
}));

const columns: DataTableColumns<RuleItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: () => t('page.notification.rule.name'), key: 'name' },
  { title: () => t('page.notification.rule.ruleType'), key: 'ruleType' },
  { title: () => t('page.notification.rule.eventType'), key: 'eventType' },
  { 
    title: () => t('page.notification.rule.status'), 
    key: 'enabled',
    render(row) {
      return h(NTag, { type: row.enabled ? 'success' : 'error', bordered: false }, { default: () => row.enabled ? t('page.notification.rule.enabled') : t('page.notification.rule.disabled') });
    }
  },
  { title: () => t('page.notification.rule.description'), key: 'description' },
  {
    title: () => t('common.action'),
    key: 'action',
    render(row) {
      return h('div', { class: 'flex gap-2' }, [
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

function handleDelete(row: RuleItem) {
  dialog.warning({
    title: t('page.notification.rule.deleteConfirmTitle'),
    content: t('page.notification.rule.deleteConfirmContent', { name: row.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const { error } = await deleteRule(row.id);
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
  fetchDependencies();
});
</script>
