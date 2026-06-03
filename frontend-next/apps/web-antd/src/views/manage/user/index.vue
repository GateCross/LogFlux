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
  Select,
  SelectOption,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import { requestClient } from '#/api/request';

defineOptions({ name: 'ManageUser' });

// --------------- types ---------------
interface UserItem {
  id: number;
  username: string;
  roles: string[];
  status: string;
  createdAt: string;
}

// --------------- placeholder API calls ---------------
async function getUserListApi() {
  return requestClient.get<UserItem[]>('/system/user/list');
}

async function createUserApi(data: {
  password: string;
  roles: string[];
  username: string;
}) {
  return requestClient.post('/system/user', data);
}

async function updateUserApi(
  id: number,
  data: { password?: string; roles?: string[]; username?: string },
) {
  return requestClient.put(`/system/user/${id}`, data);
}

async function deleteUserApi(id: number) {
  return requestClient.delete(`/system/user/${id}`);
}

// --------------- state ---------------
const loading = ref(false);
const dataSource = ref<UserItem[]>([]);
const modalVisible = ref(false);
const modalTitle = ref('Add User');
const editingId = ref<number | null>(null);
const submitting = ref(false);

const formState = reactive({
  password: '',
  roles: [] as string[],
  username: '',
});

const roleOptions = ['admin', 'editor', 'viewer'];

// --------------- columns ---------------
const columns = [
  { dataIndex: 'username', key: 'username', title: 'Username' },
  { dataIndex: 'roles', key: 'roles', title: 'Roles' },
  { dataIndex: 'status', key: 'status', title: 'Status' },
  { dataIndex: 'createdAt', key: 'createdAt', title: 'Created At' },
  { key: 'actions', title: 'Actions', width: 180 },
];

// --------------- data fetching ---------------
async function fetchData() {
  loading.value = true;
  try {
    dataSource.value = await getUserListApi();
  } catch {
    message.error('Failed to load user list');
  } finally {
    loading.value = false;
  }
}

// --------------- modal actions ---------------
function openAddModal() {
  modalTitle.value = 'Add User';
  editingId.value = null;
  formState.username = '';
  formState.password = '';
  formState.roles = [];
  modalVisible.value = true;
}

function openEditModal(record: UserItem) {
  modalTitle.value = 'Edit User';
  editingId.value = record.id;
  formState.username = record.username;
  formState.password = '';
  formState.roles = [...record.roles];
  modalVisible.value = true;
}

async function handleSubmit() {
  if (!formState.username) {
    message.warning('Username is required');
    return;
  }
  if (!editingId.value && !formState.password) {
    message.warning('Password is required for new users');
    return;
  }

  submitting.value = true;
  try {
    if (editingId.value) {
      const payload: Record<string, any> = {
        roles: formState.roles,
        username: formState.username,
      };
      if (formState.password) {
        payload.password = formState.password;
      }
      await updateUserApi(editingId.value, payload);
      message.success('User updated successfully');
    } else {
      await createUserApi({
        password: formState.password,
        roles: formState.roles,
        username: formState.username,
      });
      message.success('User created successfully');
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
    await deleteUserApi(id);
    message.success('User deleted successfully');
    await fetchData();
  } catch {
    message.error('Failed to delete user');
  }
}

// --------------- lifecycle ---------------
onMounted(() => {
  fetchData();
});
</script>

<template>
  <div class="p-5">
    <Card title="User Management">
      <template #extra>
        <Button type="primary" @click="openAddModal">Add User</Button>
      </template>

      <Table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'roles'">
            <Space>
              <Tag v-for="role in record.roles" :key="role" color="blue">
                {{ role }}
              </Tag>
            </Space>
          </template>
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
                title="Are you sure to delete this user?"
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
        <FormItem label="Username" required>
          <Input
            v-model:value="formState.username"
            placeholder="Enter username"
            :disabled="!!editingId"
          />
        </FormItem>
        <FormItem :label="editingId ? 'New Password (leave blank to keep)' : 'Password'" :required="!editingId">
          <Input.Password
            v-model:value="formState.password"
            placeholder="Enter password"
          />
        </FormItem>
        <FormItem label="Roles">
          <Select
            v-model:value="formState.roles"
            mode="multiple"
            placeholder="Select roles"
          >
            <SelectOption v-for="role in roleOptions" :key="role" :value="role">
              {{ role }}
            </SelectOption>
          </Select>
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>
