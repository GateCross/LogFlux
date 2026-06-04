<script lang="ts" setup>
import type { RoleApi } from '#/api/system/role';

import { computed, onMounted, ref } from 'vue';

import {
  Button,
  Card,
  Checkbox,
  CheckboxGroup,
  Form,
  FormItem,
  message,
  Modal,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import {
  getRoleListApi,
  updateRolePermissionsApi,
} from '#/api/system/role';

defineOptions({ name: 'ManageRole' });

const loading = ref(false);
const dataSource = ref<RoleApi.RoleItem[]>([]);

const modalVisible = ref(false);
const currentRoleId = ref<number | null>(null);
const currentRoleName = ref('');
const currentPermissions = ref<string[]>([]);
const submitting = ref(false);

const permissionGroups = [
  {
    label: '仪表盘',
    options: [{ label: '仪表盘', value: 'dashboard' }],
  },
  {
    label: '日志与安全',
    options: [
      { label: '系统日志', value: 'logs' },
      { label: 'Caddy 代理日志', value: 'logs_caddy' },
      { label: '安全升级管理', value: 'security' },
    ],
  },
  {
    label: '系统管理',
    options: [
      { label: '用户管理', value: 'manage_user' },
      { label: '角色管理', value: 'manage_role' },
      { label: '菜单管理', value: 'manage_menu' },
    ],
  },
];

const columns = [
  { dataIndex: 'id', key: 'id', title: 'ID', width: 80 },
  { dataIndex: 'name', key: 'name', title: '角色名', width: 140 },
  { dataIndex: 'displayName', key: 'displayName', title: '显示名称', width: 160 },
  { dataIndex: 'description', key: 'description', title: '描述' },
  { dataIndex: 'permissions', key: 'permissions', title: '权限', width: 260 },
  { dataIndex: 'createdAt', key: 'createdAt', title: '创建时间', width: 180 },
  { key: 'actions', title: '操作', width: 120 },
];

const modalTitle = computed(() =>
  currentRoleName.value ? `编辑权限 - ${currentRoleName.value}` : '编辑权限',
);

async function fetchData() {
  loading.value = true;
  try {
    dataSource.value = await getRoleListApi();
  } catch {
    message.error('加载角色列表失败');
  } finally {
    loading.value = false;
  }
}

function openPermissionModal(record: RoleApi.RoleItem) {
  currentRoleId.value = record.id;
  currentRoleName.value = record.displayName || record.name;
  currentPermissions.value = [...(record.permissions ?? [])];
  modalVisible.value = true;
}

async function handleSubmit() {
  if (!currentRoleId.value) return;

  submitting.value = true;
  try {
    await updateRolePermissionsApi(currentRoleId.value, currentPermissions.value);
    message.success('权限更新成功');
    modalVisible.value = false;
    await fetchData();
  } catch {
    message.error('权限更新失败');
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  void fetchData();
});
</script>

<template>
  <div class="p-5">
    <Card title="角色管理">
      <template #extra>
        <Button @click="fetchData">刷新</Button>
      </template>

      <Table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'permissions'">
            <template v-if="record.permissions?.length">
              <Tag
                v-for="permission in record.permissions"
                :key="permission"
                color="blue"
              >
                {{ permission }}
              </Tag>
            </template>
            <Tag v-else>无</Tag>
          </template>

          <template v-if="column.key === 'actions'">
            <Button
              type="link"
              size="small"
              @click="openPermissionModal(record as RoleApi.RoleItem)"
            >
              编辑权限
            </Button>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      :confirm-loading="submitting"
      :open="modalVisible"
      :title="modalTitle"
      :width="680"
      @cancel="modalVisible = false"
      @ok="handleSubmit"
    >
      <Form layout="vertical" class="mt-4">
        <FormItem label="权限列表">
          <CheckboxGroup v-model:value="currentPermissions">
            <div
              v-for="group in permissionGroups"
              :key="group.label"
              class="mb-4"
            >
              <div class="mb-2 text-sm font-medium text-gray-700">
                {{ group.label }}
              </div>
              <Space direction="vertical">
                <Checkbox
                  v-for="option in group.options"
                  :key="option.value"
                  :value="option.value"
                >
                  {{ option.label }}
                </Checkbox>
              </Space>
            </div>
          </CheckboxGroup>
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>
