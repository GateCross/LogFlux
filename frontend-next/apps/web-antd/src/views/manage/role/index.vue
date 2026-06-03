<script lang="ts" setup>
import { onMounted, reactive, ref } from 'vue';

import {
  Button,
  Card,
  Form,
  FormItem,
  Input,
  message,
  Modal,
  Popconfirm,
  Space,
  Switch,
  Table,
  Tag,
} from 'ant-design-vue';

import {
  getRoleListApi,
  updateRolePermissionsApi,
} from '#/api/system/role';

import { requestClient } from '#/api/request';

defineOptions({ name: 'ManageRole' });

// --------------- types ---------------
interface RoleItem {
  id: number;
  name: string;
  description: string;
  status: string;
  permissions: string[];
}

// --------------- placeholder API calls ---------------
async function createRoleApi(data: {
  description: string;
  name: string;
  permissions: string[];
}) {
  return requestClient.post('/role', data);
}

async function updateRoleApi(
  id: number,
  data: { description?: string; name?: string; permissions?: string[] },
) {
  return requestClient.put(`/role/${id}`, data);
}

async function deleteRoleApi(id: number) {
  return requestClient.delete(`/role/${id}`);
}

// --------------- state ---------------
const loading = ref(false);
const dataSource = ref<RoleItem[]>([]);
const modalVisible = ref(false);
const modalTitle = ref('Add Role');
const editingId = ref<number | null>(null);
const submitting = ref(false);

const formState = reactive({
  description: '',
  name: '',
  permissions: [] as string[],
});

const permissionOptions = [
  'user:read',
  'user:write',
  'role:read',
  'role:write',
  'menu:read',
  'menu:write',
  'log:read',
  'log:write',
  'system:config',
];

// --------------- columns ---------------
const columns = [
  { dataIndex: 'name', key: 'name', title: 'Name' },
  { dataIndex: 'description', key: 'description', title: 'Description' },
  { dataIndex: 'status', key: 'status', title: 'Status' },
  { key: 'actions', title: 'Actions', width: 220 },
];

// --------------- data fetching ---------------
async function fetchData() {
  loading.value = true;
  try {
    dataSource.value = await getRoleListApi();
  } catch {
    message.error('Failed to load role list');
  } finally {
    loading.value = false;
  }
}

// --------------- modal actions ---------------
function openAddModal() {
  modalTitle.value = 'Add Role';
  editingId.value = null;
  formState.name = '';
  formState.description = '';
  formState.permissions = [];
  modalVisible.value = true;
}

function openEditModal(record: RoleItem) {
  modalTitle.value = 'Edit Role';
  editingId.value = record.id;
  formState.name = record.name;
  formState.description = record.description;
  formState.permissions = [...(record.permissions ?? [])];
  modalVisible.value = true;
}

async function handleSubmit() {
  if (!formState.name) {
    message.warning('Role name is required');
    return;
  }

  submitting.value = true;
  try {
    if (editingId.value) {
      await updateRoleApi(editingId.value, {
        description: formState.description,
        name: formState.name,
        permissions: formState.permissions,
      });
      // also sync permissions via dedicated endpoint
      await updateRolePermissionsApi(editingId.value, formState.permissions);
      message.success('Role updated successfully');
    } else {
      await createRoleApi({
        description: formState.description,
        name: formState.name,
        permissions: formState.permissions,
      });
      message.success('Role created successfully');
    }
    modalVisible.value = false;
    await fetchData();
  } catch {
    message.error('Operation failed');
  } finally {
    submitting.value = false;
  }
}

async function handleDelete(id: number) {
  try {
    await deleteRoleApi(id);
    message.success('Role deleted successfully');
    await fetchData();
  } catch {
    message.error('Failed to delete role');
  }
}

function handlePermissionChange(checked: boolean, permission: string) {
  if (checked) {
    formState.permissions.push(permission);
  } else {
    const idx = formState.permissions.indexOf(permission);
    if (idx >= 0) {
      formState.permissions.splice(idx, 1);
    }
  }
}

// --------------- lifecycle ---------------
onMounted(() => {
  fetchData();
});
</script>

<template>
  <div class="p-5">
    <Card title="Role Management">
      <template #extra>
        <Button type="primary" @click="openAddModal">Add Role</Button>
      </template>

      <Table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <Tag :color="record.status === 'active' ? 'green' : 'red'">
              {{ record.status }}
            </Tag>
          </template>
          <template v-if="column.key === 'actions'">
            <Space>
              <Button type="link" size="small" @click="openEditModal(record)">
                Edit
              </Button>
              <Popconfirm
                title="Are you sure to delete this role?"
                ok-text="Yes"
                cancel-text="No"
                @confirm="handleDelete(record.id)"
              >
                <Button type="link" size="small" danger>Delete</Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      :open="modalVisible"
      :title="modalTitle"
      :confirm-loading="submitting"
      @cancel="modalVisible = false"
      @ok="handleSubmit"
    >
      <Form layout="vertical" style="margin-top: 16px;">
        <FormItem label="Role Name" required>
          <Input
            v-model:value="formState.name"
            placeholder="Enter role name"
          />
        </FormItem>
        <FormItem label="Description">
          <Input.TextArea
            v-model:value="formState.description"
            placeholder="Enter role description"
            :rows="3"
          />
        </FormItem>
        <FormItem label="Permissions">
          <div style="display: flex; flex-wrap: wrap; gap: 12px;">
            <div
              v-for="perm in permissionOptions"
              :key="perm"
              style="display: flex; align-items: center; gap: 6px;"
            >
              <Switch
                :checked="formState.permissions.includes(perm)"
                size="small"
                @change="(checked: boolean) => handlePermissionChange(checked, perm)"
              />
              <span>{{ perm }}</span>
            </div>
          </div>
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>
