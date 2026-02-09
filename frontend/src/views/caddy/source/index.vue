<template>
  <div class="h-full">
    <n-card title="日志源管理" :bordered="false" class="h-full rounded-8px shadow-sm">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          新增日志源
        </n-button>
      </template>

      <n-data-table
        remote
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        :row-key="row => row.id"
        class="h-full"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" class="w-600px">
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="100">
        <n-form-item label="名称" path="name">
          <n-input v-model:value="formModel.name" placeholder="例如：本地 Caddy 日志" />
        </n-form-item>
        <n-form-item label="路径" path="path">
          <n-input v-model:value="formModel.path" placeholder="/var/log/caddy/access.log 或 /var/log/caddy" />
        </n-form-item>
        <n-form-item label="类型" path="type">
          <n-select v-model:value="formModel.type" :options="typeOptions" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="扫描间隔(秒)" path="scanInterval">
          <n-input-number v-model:value="formModel.scanInterval" :min="1" :max="3600" :step="1" />
          <div class="text-xs text-gray-500 mt-1">默认 60 秒</div>
        </n-form-item>
        <n-form-item label="启用" path="enabled">
          <n-switch v-model:value="formModel.enabled" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">保存</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h, computed } from 'vue';
import { NButton, NTag, NSwitch, useMessage, useDialog } from 'naive-ui';
import type { DataTableColumns, FormInst, FormRules, PaginationProps } from 'naive-ui';
import { createLogSource, deleteLogSource, fetchLogSourceList, updateLogSource } from '@/service/api/log-source';
import type { LogSourceItem } from '@/service/api/log-source';

const message = useMessage();
const dialog = useDialog();

const loading = ref(false);
const tableData = ref<LogSourceItem[]>([]);

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
});

const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitting = ref(false);
const formRef = ref<FormInst | null>(null);

const formModel = ref({
  id: 0,
  name: '',
  path: '',
  type: 'caddy',
  scanInterval: 60,
  enabled: true
});

const isEdit = computed(() => modalType.value === 'edit');
const modalTitle = computed(() => (isEdit.value ? '编辑日志源' : '新增日志源'));

const rules: FormRules = {
  path: { required: true, message: '请输入日志文件/目录路径', trigger: 'blur' },
  scanInterval: { type: 'number', min: 1, message: '请输入大于 0 的扫描间隔', trigger: 'blur' }
};

const typeOptions = [
  { label: 'Caddy代理日志', value: 'caddy' },
  { label: 'Caddy后台日志', value: 'caddy_runtime' },
  { label: 'Nginx', value: 'nginx' },
  { label: '其他', value: 'other' }
];

const columns: DataTableColumns<LogSourceItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '名称', key: 'name', minWidth: 140 },
  { title: '路径', key: 'path', minWidth: 260 },
  { title: '扫描间隔(秒)', key: 'scanInterval', width: 140 },
  {
    title: '类型',
    key: 'type',
    width: 120,
    render(row) {
      const labelMap: Record<string, string> = {
        caddy: 'Caddy代理日志',
        caddy_runtime: 'Caddy后台日志',
        nginx: 'Nginx',
        other: '其他'
      };
      return h(NTag, { type: 'info', bordered: false }, { default: () => labelMap[row.type] || row.type });
    }
  },
  {
    title: '启用',
    key: 'enabled',
    width: 100,
    render(row) {
      return h(NSwitch, {
        value: row.enabled,
        onUpdateValue: value => handleToggle(row, value)
      });
    }
  },
  { title: '创建时间', key: 'createdAt', width: 180 },
  {
    title: '操作',
    key: 'action',
    width: 160,
    render(row) {
      return h('div', { class: 'flex gap-2' }, [
        h(NButton, { size: 'small', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
        h(
          NButton,
          {
            size: 'small',
            type: 'error',
            onClick: () => handleDelete(row)
          },
          { default: () => '删除' }
        )
      ]);
    }
  }
];

async function fetchData() {
  loading.value = true;
  try {
    const { data, error } = await fetchLogSourceList({
      page: pagination.page as number,
      pageSize: pagination.pageSize as number
    });
    if (!error && data) {
      tableData.value = data.list || [];
      pagination.itemCount = data.total || 0;
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
    path: '',
    type: 'caddy',
    scanInterval: 60,
    enabled: true
  };
  showModal.value = true;
}

function handleEdit(row: LogSourceItem) {
  modalType.value = 'edit';
  formModel.value = {
    id: row.id,
    name: row.name,
    path: row.path,
    type: row.type,
    scanInterval: row.scanInterval,
    enabled: row.enabled
  };
  showModal.value = true;
}

async function handleSubmit() {
  await formRef.value?.validate();
  submitting.value = true;
  try {
    if (modalType.value === 'add') {
      const { error } = await createLogSource({
        name: formModel.value.name,
        path: formModel.value.path,
        type: formModel.value.type,
        scanInterval: formModel.value.scanInterval
      });
      if (!error) {
        message.success('新增成功');
        showModal.value = false;
        fetchData();
      }
      return;
    }

    const { error } = await updateLogSource(formModel.value.id, {
      name: formModel.value.name,
      path: formModel.value.path,
      scanInterval: formModel.value.scanInterval,
      enabled: formModel.value.enabled
    });
    if (!error) {
      message.success('更新成功');
      showModal.value = false;
      fetchData();
    }
  } finally {
    submitting.value = false;
  }
}

async function handleToggle(row: LogSourceItem, value: boolean) {
  const { error } = await updateLogSource(row.id, { enabled: value });
  if (error) {
    message.error('更新失败');
    return;
  }
  row.enabled = value;
}

function handleDelete(row: LogSourceItem) {
  dialog.warning({
    title: '确认删除',
    content: `确定删除日志源 "${row.name}" 吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      const { error } = await deleteLogSource(row.id);
      if (!error) {
        message.success('删除成功');
        fetchData();
      }
    }
  });
}

function handlePageChange(page: number) {
  pagination.page = page;
  fetchData();
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize;
  pagination.page = 1;
  fetchData();
}

onMounted(() => {
  fetchData();
});
</script>
