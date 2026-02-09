<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NButton, NCard, NDataTable, NModal, NCheckboxGroup, NCheckbox, NSpace, NForm, NFormItem, useMessage } from 'naive-ui';
import { fetchGetRoleList, fetchUpdateRolePermissions } from '@/service/api/role';
import { $t } from '@/locales';

const message = useMessage();

const loading = ref(false);
const data = ref<Api.Role.RoleItem[]>([]);

const showModal = ref(false);
const currentRoleId = ref<number | null>(null);
const currentPermissions = ref<string[]>([]);
const submitLoading = ref(false);

// 权限分组选项
const permissionGroups = [
  {
    label: '仪表盘',
    options: [
      { label: '仪表盘', value: 'dashboard' }
    ]
  },
  {
    label: '日志管理',
    options: [
      { label: '系统日志（/caddy/system-log）', value: 'logs' },
      { label: 'Caddy代理日志（/caddy/log）', value: 'logs_caddy' }
    ]
  },
  {
    label: '系统管理',
    options: [
      { label: '用户管理', value: 'manage_user' },
      { label: '角色管理', value: 'manage_role' }
    ]
  }
];

const columns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '角色名', key: 'name', width: 150 },
  { title: '显示名称', key: 'displayName', width: 150 },
  { title: '描述', key: 'description' },
  { title: '创建时间', key: 'createdAt', width: 180 },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render(row: Api.Role.RoleItem) {
      return (
        <NButton size="small" type="primary" onClick={() => handleEditPermissions(row)}>
          编辑权限
        </NButton>
      );
    }
  }
];

async function init() {
  loading.value = true;
  try {
    const { data: resp } = await fetchGetRoleList();
    if (resp?.list) {
      data.value = resp.list;
    }
  } finally {
    loading.value = false;
  }
}

function handleEditPermissions(row: Api.Role.RoleItem) {
  currentRoleId.value = row.id;
  // Make a copy of permissions
  currentPermissions.value = [...(row.permissions || [])];
  showModal.value = true;
}

async function handleSubmit() {
  if (!currentRoleId.value) return;

  submitLoading.value = true;
  try {
    const { error } = await fetchUpdateRolePermissions(currentRoleId.value, currentPermissions.value);
    if (!error) {
      message.success('权限更新成功');
      showModal.value = false;
      init(); // Refresh list
    }
  } finally {
    submitLoading.value = false;
  }
}

onMounted(() => {
  init();
});
</script>

<template>
  <div class="h-full overflow-hidden">
    <NCard title="角色管理" :bordered="false" class="h-full rounded-8px shadow-sm">
      <NDataTable
        :columns="columns"
        :data="data"
        :loading="loading"
        :row-key="row => row.id"
        flex-height
        class="h-full"
      />
    </NCard>

    <NModal v-model:show="showModal" preset="card" title="编辑权限" class="w-700px">
      <NForm>
        <NFormItem label="权限列表">
          <NCheckboxGroup v-model:value="currentPermissions">
            <div v-for="group in permissionGroups" :key="group.label" class="mb-4">
              <div class="text-sm font-medium mb-2 text-gray-700">{{ group.label }}</div>
              <NSpace vertical>
                <NCheckbox
                  v-for="opt in group.options"
                  :key="opt.value"
                  :value="opt.value"
                  :label="opt.label"
                  class="ml-4"
                />
              </NSpace>
            </div>
          </NCheckboxGroup>
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showModal = false">取消</NButton>
          <NButton type="primary" :loading="submitLoading" @click="handleSubmit">保存</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
