<template>
  <div class="h-full overflow-hidden flex flex-col">
    <n-card title="定时任务" class="h-full flex-1" content-style="display: flex; flex-direction: column; overflow: hidden;">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <icon-ic-round-plus />
          </template>
          新增任务
        </n-button>
      </template>

      <div class="h-full flex flex-col">
        <n-data-table
          :columns="columns"
          :data="data"
          :loading="loading"
          :pagination="pagination"
          remote
          class="flex-1"
          flex-height
          @update:page="handlePageChange"
        />
      </div>
    </n-card>

    <n-modal v-model:show="showModal" preset="card" :title="modalType === 'add' ? '新增任务' : '编辑任务'" class="w-600px">
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="80">
        <n-form-item label="任务名称" path="name">
          <n-input v-model:value="formModel.name" placeholder="请输入任务名称" />
        </n-form-item>
        <n-form-item label="Cron表达式" path="schedule">
          <n-input v-model:value="formModel.schedule" placeholder="例如: 0/5 * * * * ?" />
          <n-text depth="3" class="ml-2 text-12px">支持秒级 (e.g., * * * * * *)</n-text>
        </n-form-item>
        <n-form-item label="执行脚本" path="script">
          <n-input
            v-model:value="formModel.script"
            type="textarea"
            placeholder="请输入Shell脚本"
            :autosize="{ minRows: 3, maxRows: 10 }"
          />
        </n-form-item>
        <n-form-item label="超时时间" path="timeout">
          <n-input-number v-model:value="formModel.timeout" placeholder="秒" />
          <span class="ml-2">秒</span>
        </n-form-item>
        <n-form-item label="状态" path="status">
          <n-switch
            v-model:value="formModel.status"
            :checked-value="1"
            :unchecked-value="0"
          >
            <template #checked>启用</template>
            <template #unchecked>禁用</template>
          </n-switch>
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</n-button>
        </div>
      </template>
    </n-modal>

    <!-- 日志抽屉 -->
    <n-drawer v-model:show="showLogDrawer" width="800" placement="right">
      <n-drawer-content title="执行日志">
        <cron-log-list :task-id="currentTaskId" />
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue';
import { NButton, NTag, NSpace, NPopconfirm, useMessage, type DataTableColumns, type FormInst } from 'naive-ui';
import { fetchCronTaskList, createCronTask, updateCronTask, deleteCronTask, triggerCronTask, type CronTask } from '@/service/api/cron';
import CronLogList from './log.vue'; 

// Components are auto-imported in this project usually, but imports are safer.
// Using standard naive-ui components.

const message = useMessage();
const formRef = ref<FormInst | null>(null);

// Data
const data = ref<CronTask[]>([]);
const loading = ref(false);
const pagination = reactive({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  onChange: (page: number) => {
    pagination.page = page;
    getData();
  }
});

// Modal
const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitLoading = ref(false);
const formModel = reactive({
  id: 0,
  name: '',
  schedule: '',
  script: '',
  status: 1,
  timeout: 60
});

const rules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  schedule: [{ required: true, message: '请输入Cron表达式', trigger: 'blur' }],
  script: [{ required: true, message: '请输入执行脚本', trigger: 'blur' }]
};

// Log Drawer
const showLogDrawer = ref(false);
const currentTaskId = ref(0);

const columns: DataTableColumns<CronTask> = [
  { title: '任务名称', key: 'name', width: 150 },
  { title: 'Cron表达式', key: 'schedule', width: 150, render: (row) => h(NTag, { type: 'info', size: 'small' }, { default: () => row.schedule }) },
  { 
    title: '状态', 
    key: 'status', 
    width: 100,
    render: (row) => h(NTag, { type: row.status === 1 ? 'success' : 'error', size: 'small' }, { default: () => row.status === 1 ? '已启用' : '已禁用' })
  },
  { title: '下次执行时间', key: 'nextRun', width: 180 },
  { title: '上次更新', key: 'updatedAt', width: 180 },
  {
    title: '操作',
    key: 'actions',
    width: 250,
    render(row) {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, {
            size: 'small',
            type: 'warning',
            secondary: true,
            onClick: () => handleTrigger(row)
          }, { default: () => '手动执行' }),
          h(NButton, {
            size: 'small',
            onClick: () => handleEdit(row)
          }, { default: () => '编辑' }),
          h(NButton, {
            size: 'small',
            secondary: true,
            onClick: () => openLog(row.id)
          }, { default: () => '日志' }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row.id)
          }, {
            trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, { default: () => '删除' }),
            default: () => '确认删除该任务吗？'
          })
        ]
      });
    }
  }
];

async function getData() {
  loading.value = true;
  try {
    const { data: res } = await fetchCronTaskList({
      page: pagination.page,
      pageSize: pagination.pageSize
    });
    if (res) {
      data.value = res.list;
      pagination.itemCount = res.total;
    }
  } finally {
    loading.value = false;
  }
}

function handleAdd() {
  modalType.value = 'add';
  Object.assign(formModel, {
    id: 0,
    name: '',
    schedule: '',
    script: '',
    status: 1,
    timeout: 60
  });
  showModal.value = true;
}

function handleEdit(row: CronTask) {
  modalType.value = 'edit';
  Object.assign(formModel, row);
  showModal.value = true;
}

async function handleSubmit() {
  await formRef.value?.validate();
  submitLoading.value = true;
  try {
    if (modalType.value === 'add') {
      await createCronTask(formModel);
      message.success('创建成功');
    } else {
      await updateCronTask(formModel.id, formModel);
      message.success('更新成功');
    }
    showModal.value = false;
    getData();
  } finally {
    submitLoading.value = false;
  }
}

async function handleDelete(id: number) {
  await deleteCronTask(id);
  message.success('删除成功');
  getData();
}

async function handleTrigger(row: CronTask) {
  await triggerCronTask(row.id);
  message.success('已触发手动执行');
}

function openLog(id: number) {
  currentTaskId.value = id;
  showLogDrawer.value = true;
}

function handlePageChange(page: number) {
  pagination.page = page;
  getData();
}

onMounted(() => {
  getData();
});
</script>
