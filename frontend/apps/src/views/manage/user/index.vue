<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import {
  Button,
  Card,
  Form,
  Input,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';
import {
  createUserApi,
  deleteUserApi,
  getUserListApi,
  toggleUserStatusApi,
  updateUserApi,
} from '#/api/system/user';
import { getRoleListApi } from '#/api/system/role';

defineOptions({ name: 'ManageUser' });

interface UserItem {
  id: number;
  username: string;
  roles: string[];
  status: number;
  createdAt: string;
}

const loading = ref(false);
const dataSource = ref<UserItem[]>([]);
const total = ref(0);
const searchUsername = ref('');

const pagination = reactive({
  current: 1,
  pageSize: 20,
  showSizeChanger: true,
  pageSizeOptions: ['10', '20', '50', '100'],
  showTotal: (t: number) => `共 ${t} 条`,
});

const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitLoading = ref(false);
const formState = reactive({
  id: 0,
  username: '',
  password: '',
  roles: [] as string[],
});

const roleOptions = ref<Array<{ label: string; value: string }>>([]);

const roleMap: Record<string, string> = {
  admin: '管理员',
  analyst: '分析师',
  viewer: '访客',
};

const columns = [
  { dataIndex: 'id', key: 'id', title: 'ID', width: 80 },
  { dataIndex: 'username', key: 'username', title: '用户名', width: 150 },
  { key: 'roles', title: '角色', width: 200 },
  { key: 'status', title: '状态', width: 100 },
  { dataIndex: 'createdAt', key: 'createdAt', title: '创建时间', width: 180 },
  { key: 'actions', title: '操作', width: 200 },
];

async function fetchRoleOptions() {
  try {
    const list = await getRoleListApi();
    if (Array.isArray(list)) {
      roleOptions.value = list.map((role: any) => ({
        label: role.displayName || role.name,
        value: role.name,
      }));
    }
  } catch {
    // error handled by interceptor
  }
}

async function fetchData() {
  loading.value = true;
  try {
    const params: Record<string, any> = {
      page: pagination.current,
      pageSize: pagination.pageSize,
    };
    if (searchUsername.value) {
      params.username = searchUsername.value;
    }
    const data = await getUserListApi(params);
    dataSource.value = data?.list ?? [];
    total.value = data?.total ?? 0;
  } catch {
    message.error('加载失败');
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  pagination.current = 1;
  fetchData();
}

function handleTableChange(pag: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchData();
}

function handleAdd() {
  modalType.value = 'add';
  formState.username = '';
  formState.password = '';
  formState.roles = ['viewer'];
  showModal.value = true;
}

function handleEdit(record: UserItem) {
  modalType.value = 'edit';
  formState.id = record.id;
  formState.username = record.username;
  formState.password = '';
  formState.roles = [...record.roles];
  showModal.value = true;
}

async function handleDelete(id: number) {
  try {
    await deleteUserApi(id);
    message.success('删除成功');
    fetchData();
  } catch {
    message.error('删除失败');
  }
}

async function handleToggleStatus(record: UserItem) {
  try {
    await toggleUserStatusApi(record.id);
    message.success(record.status === 1 ? '用户已冻结' : '用户已解冻');
    fetchData();
  } catch {
    message.error('操作失败');
  }
}

async function handleSubmit() {
  if (!formState.username) {
    message.warning('请输入用户名');
    return;
  }
  if (modalType.value === 'add' && !formState.password) {
    message.warning('请输入密码');
    return;
  }

  submitLoading.value = true;
  try {
    if (modalType.value === 'add') {
      await createUserApi({
        username: formState.username,
        password: formState.password,
        roles: formState.roles,
      });
      message.success('新增成功');
    } else {
      const payload: Record<string, any> = { roles: formState.roles };
      if (formState.password) {
        payload.password = formState.password;
      }
      await updateUserApi(formState.id, payload);
      message.success('编辑成功');
    }
    showModal.value = false;
    fetchData();
  } catch {
    message.error('操作失败');
  } finally {
    submitLoading.value = false;
  }
}

onMounted(() => {
  fetchRoleOptions();
  fetchData();
});
</script>

<template>
  <div class="h-full overflow-x-hidden overflow-y-auto">
    <Card title="用户管理" :bordered="false" class="h-full rounded-lg shadow-sm">
      <template #extra>
        <Space>
          <Input
            v-model:value="searchUsername"
            placeholder="搜索用户名"
            allow-clear
            @press-enter="handleSearch"
          />
          <Button type="primary" @click="handleSearch">搜索</Button>
          <Button type="primary" ghost @click="handleAdd">新增用户</Button>
        </Space>
      </template>

      <Table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        :pagination="{ ...pagination, total }"
        row-key="id"
        size="middle"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'roles'">
            <Space>
              <Tag
                v-for="role in record.roles"
                :key="role"
                :color="role === 'admin' ? 'green' : 'blue'"
              >
                {{ roleMap[role] || role }}
              </Tag>
            </Space>
          </template>
          <template v-if="column.key === 'status'">
            <Tag :color="record.status === 1 ? 'green' : 'red'">
              {{ record.status === 1 ? '启用' : '禁用' }}
            </Tag>
          </template>
          <template v-if="column.key === 'actions'">
            <Space>
              <Button type="link" size="small" @click="handleEdit(record as UserItem)">
                编辑
              </Button>
              <Button
                type="link"
                size="small"
                :style="{ color: record.status === 1 ? '#faad14' : '#52c41a' }"
                @click="handleToggleStatus(record as UserItem)"
              >
                {{ record.status === 1 ? '冻结' : '解冻' }}
              </Button>
              <Popconfirm
                title="确认永久删除该用户吗？此操作无法恢复！"
                ok-text="确认"
                cancel-text="取消"
                @confirm="handleDelete(record.id)"
              >
                <Button type="link" size="small" danger>删除</Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>

    <Modal
      :open="showModal"
      :title="modalType === 'add' ? '新增用户' : '编辑用户'"
      :confirm-loading="submitLoading"
      @cancel="showModal = false"
      @ok="handleSubmit"
    >
      <Form layout="vertical" style="margin-top: 16px;">
        <Form.Item label="用户名" required>
          <Input
            v-model:value="formState.username"
            :disabled="modalType === 'edit'"
            placeholder="请输入用户名"
          />
        </Form.Item>
        <Form.Item
          :label="modalType === 'edit' ? '新密码（留空则不修改）' : '密码'"
          :required="modalType === 'add'"
        >
          <Input.Password
            v-model:value="formState.password"
            placeholder="请输入密码"
          />
        </Form.Item>
        <Form.Item label="角色">
          <Select
            v-model:value="formState.roles"
            mode="multiple"
            :options="roleOptions"
            placeholder="请选择角色"
          />
        </Form.Item>
      </Form>
    </Modal>
  </div>
</template>
