<template>
  <div class="h-full overflow-hidden">
    <n-card title="用户管理" :bordered="false" class="h-full rounded-8px shadow-sm">
      <div class="flex-col h-full">
        <n-space class="mb-4" justify="space-between">
          <n-input v-model:value="searchParams.username" placeholder="搜索用户名" clearable @keyup.enter="handleSearch">
            <template #prefix>
              <icon-ic-round-search class="text-16px" />
            </template>
          </n-input>
          <n-space>
            <n-button type="primary" @click="handleSearch">
              <template #icon>
                <icon-ic-round-search />
              </template>
              搜索
            </n-button>
            <n-button type="primary" ghost @click="handleAdd">
              <template #icon>
                <icon-ic-round-plus />
              </template>
              新增用户
            </n-button>
          </n-space>
        </n-space>
        
        <n-data-table
          remote
          :columns="columns"
          :data="tableData"
          :loading="loading"
          :pagination="pagination"
          :row-key="row => row.id"
          class="flex-1-hidden"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </n-card>

    <n-modal v-model:show="showModal" :title="modalType === 'add' ? '新增用户' : '编辑用户'" preset="card" class="w-600px">
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="80">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="formModel.username" :disabled="modalType === 'edit'" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="formModel.password"
            type="password"
            show-password-on="mousedown"
            :placeholder="modalType === 'add' ? '请输入密码' : '留空则不修改密码'"
          />
        </n-form-item>
        <n-form-item label="角色" path="roles">
          <n-select v-model:value="formModel.roles" multiple :options="roleOptions" placeholder="请选择角色" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="closeModal">取消</n-button>
          <n-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from 'vue';
import { NButton, NPopconfirm, NTag, useMessage } from 'naive-ui';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { request } from '@/service/request';

interface User {
  id: number;
  username: string;
  roles: string[];
  status: number; // 1=启用, 0=禁用
  createdAt: string;
}

const message = useMessage();
const loading = ref(false);
const tableData = ref<User[]>([]);
const searchParams = reactive({
  username: ''
});

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  itemCount: 0,
  onChange: (page: number) => {
    pagination.page = page;
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.pageSize = pageSize;
    pagination.page = 1;
  }
});

const columns: DataTableColumns<User> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '用户名', key: 'username', width: 150 },
  {
    title: '角色',
    key: 'roles',
    render(row) {
      const roleMap: Record<string, string> = {
        admin: '管理员',
        analyst: '分析师',
        viewer: '访客'
      };
      return row.roles.map(role => {
        return h(
          NTag,
          {
            style: { marginRight: '6px' },
            type: role === 'admin' ? 'success' : 'info',
            bordered: false,
            size: 'small'
          },
          { default: () => roleMap[role] || role }
        );
      });
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render(row) {
      return h(
        NTag,
        {
          type: row.status === 1 ? 'success' : 'error',
          bordered: false,
          size: 'small'
        },
        { default: () => (row.status === 1 ? '启用' : '禁用') }
      );
    }
  },
  { title: '创建时间', key: 'createdAt', width: 180 },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    render(row) {
      return h(
        'div',
        { class: 'flex gap-2' },
        [
          h(
            NButton,
            {
              size: 'small',
              type: 'primary',
              secondary: true,
              onClick: () => handleEdit(row)
            },
            { default: () => '编辑' }
          ),
          h(
            NButton,
            {
              size: 'small',
              type: row.status === 1 ? 'warning' : 'success',
              secondary: true,
              onClick: () => handleToggleStatus(row)
            },
            { default: () => (row.status === 1 ? '冻结' : '解冻') }
          ),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => handleDelete(row.id)
            },
            {
              default: () => '确认永久删除该用户吗？此操作无法恢复！',
              trigger: () =>
                h(
                  NButton,
                  {
                    size: 'small',
                    type: 'error',
                    secondary: true
                  },
                  { default: () => '删除' }
                )
            }
          )
        ]
      );
    }
  }
];

// Modal
const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const submitLoading = ref(false);
const formRef = ref<any>(null);
const formModel = reactive({
  id: 0,
  username: '',
  password: '',
  roles: [] as string[]
});

const rules = {
  username: { required: true, message: '请输入用户名', trigger: 'blur' },
  roles: { required: true, type: 'array', message: '请选择角色', trigger: 'change' }
  // password required check is manual based on mode
};

const roleOptions = ref<Array<{ label: string; value: string }>>([]);

// Fetch role options from API
async function fetchRoleOptions() {
  try {
    const { data } = await request<any>({ url: '/api/role/list' });
    if (data?.list) {
      roleOptions.value = data.list.map((role: any) => ({
        label: role.displayName,
        value: role.name
      }));
    }
  } catch (error) {
    console.error('Failed to fetch role options:', error);
  }
}

async function fetchData() {
  loading.value = true;
  try {
    const { data, error } = await request<any>({
      url: '/api/user/list',
      params: {
        page: pagination.page,
        pageSize: pagination.pageSize,
        username: searchParams.username
      }
    });
    console.log('User List Response:', data, error);
    if (data) {
      tableData.value = data.list || [];
      pagination.itemCount = data.total || 0;
    }
    if (error) {
      message.error('加载失败: ' + JSON.stringify(error));
    }
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  pagination.page = 1;
  fetchData();
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

function handleAdd() {
  modalType.value = 'add';
  formModel.id = 0;
  formModel.username = '';
  formModel.password = '';
  formModel.roles = ['viewer'];
  showModal.value = true;
}

function handleEdit(row: User) {
  modalType.value = 'edit';
  formModel.id = row.id;
  formModel.username = row.username;
  formModel.password = ''; // Don't fill password
  formModel.roles = [...row.roles];
  showModal.value = true;
}

async function handleDelete(id: number) {
  try {
    await request({ url: `/api/user/${id}`, method: 'delete' });
    message.success('删除成功');
    fetchData();
  } catch (error) {
    // Error handled by interceptor
  }
}

async function handleToggleStatus(row: User) {
  try {
    const { error } = await request({
      url: `/api/user/${row.id}/status`,
      method: 'put'
    });
    if (!error) {
      message.success(row.status === 1 ? '用户已冻结' : '用户已解冻');
      fetchData();
    }
  } catch (error) {
    // Error handled by interceptor
  }
}

function closeModal() {
  showModal.value = false;
}

async function handleSubmit() {
  formRef.value?.validate(async (errors: any) => {
    if (!errors) {
      if (modalType.value === 'add' && !formModel.password) {
        message.error('请输入密码');
        return;
      }
      
      submitLoading.value = true;
      try {
        let resp;
        if (modalType.value === 'add') {
          resp = await request({
            url: '/api/user',
            method: 'post',
            data: {
              username: formModel.username,
              password: formModel.password,
              roles: formModel.roles
            }
          });
        } else {
          resp = await request({
            url: `/api/user/${formModel.id}`,
            method: 'put',
            data: {
              password: formModel.password || undefined,
              roles: formModel.roles
            }
          });
        }
        
        // Check if request was successful
        if (resp && !resp.error) {
          message.success(modalType.value === 'add' ? '新增成功' : '编辑成功');
          closeModal();
          fetchData();
        } else if (resp && resp.error) {
          message.error(resp.error.msg || '操作失败');
        }
      } catch (error: any) {
        message.error(error.message || '操作失败');
      } finally {
        submitLoading.value = false;
      }
    }
  });
}

onMounted(() => {
  fetchRoleOptions();
  fetchData();
});
</script>

<style scoped></style>
