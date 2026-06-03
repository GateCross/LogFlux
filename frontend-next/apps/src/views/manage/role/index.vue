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
const modalTitle = ref('新增角色');
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
  { dataIndex: 'name', key: 'name', title: '名称' },
  { dataIndex: 'description', key: 'description', title: '描述' },
  { dataIndex: 'status', key: 'status', title: '状态' },
  { key: 'actions', title: '操作', width: 220 },
];

// --------------- data fetching ---------------
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

// --------------- modal actions ---------------
function openAddModal() {
  modalTitle.value = '新增角色';
  editingId.value = null;
  formState.name = '';
  formState.description = '';
  formState.permissions = [];
  modalVisible.value = true;
}

function open编辑Modal(record: RoleItem) {
  modalTitle.value = '编辑角色';
  editingId.value = record.id;
  formState.name = record.name;
  formState.description = record.description;
  formState.permissions = [...(record.permissions ?? [])];
  modalVisible.value = true;
}

async function handleSubmit() {
  if (!formState.name) {
    message.warning('请输入角色名');
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
      message.success('角色更新成功');
    } else {
      await createRoleApi({
        description: formState.description,
        name: formState.name,
        permissions: formState.permissions,
      });
      message.success('角色创建成功');
    }
    modalVisible.value = false;
    await fetchData();
  } catch {
    message.error('操作失败');
  } finally {
    submitting.value = false;
  }
}

async function handle删除(id: number) {
  try {
    await deleteRoleApi(id);
    message.success('角色删除成功');
    await fetchData();
  } catch {
    message.error('删除角色失败');
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
    <Card title="角色管理">
      <template #extra>
        <Button type="primary" @click="openAddModal">新增角色</Button>
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
              <Button type="link" size="small" @click="open编辑Modal(record)">
                编辑
              </Button>
              <Popconfirm
                title="确认删除此角色？"
                ok-text="确认"
                cancel-text="取消"
                @confirm="handle删除(record.id)"
              >
                <Button type="link" size="small" danger>删除</Button>
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
        <FormItem label="Role 名称" required>
          <Input
            v-model:value="formState.name"
            placeholder="请输入角色名"
          />
        </FormItem>
        <FormItem label="描述">
          <Input.TextArea
            v-model:value="formState.description"
            placeholder="请输入角色描述"
            :rows="3"
          />
        </FormItem>
        <FormItem label="权限">
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
