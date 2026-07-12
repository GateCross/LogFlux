<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { VxeTableGridColumns } from '#/adapter/vxe-table';

import type { UserManageApi } from '#/api/system/user';

import { nextTick, onMounted, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import {
  Alert,
  Button,
  message,
  Modal,
  Space,
  Tag,
} from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  createUserApi,
  deleteUserApi,
  getUserListApi,
  toggleUserStatusApi,
  updateUserApi,
} from '#/api/system/user';
import { getRoleListApi } from '#/api/system/role';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { apiErrorMessage } from '#/utils/api-error-message';
import { toPageParams } from '#/utils/pagination';

import UserActionCell from './components/UserActionCell.vue';

import {
  createAdminCrudGridOptions,
  toVxeProxyResult,
  useAdminCrudForm,
} from '../_pattern';

defineOptions({ name: 'ManageUser' });

type UserItem = UserManageApi.UserItem;

interface UserFormValues extends Record<string, unknown> {
  username: string;
  password: string;
  roles: string[];
}

const ROLE_LABEL: Record<string, string> = {
  admin: '管理员',
  analyst: '分析师',
  viewer: '访客',
};

const queryClient = useQueryClient();
const roleOptions = ref<Array<{ label: string; value: string }>>([]);
const listErrorMessage = ref<string | null>(null);

const querySchema: VbenFormSchema[] = [
  {
    component: 'Input',
    fieldName: 'username',
    label: '用户名',
    componentProps: {
      allowClear: true,
      placeholder: '搜索用户名',
    },
  },
];

const columns: VxeTableGridColumns<UserItem> = [
  { field: 'id', title: 'ID', width: 80 },
  { field: 'username', title: '用户名', minWidth: 160 },
  {
    field: 'roles',
    title: '角色',
    minWidth: 200,
    slots: { default: 'roles' },
  },
  {
    field: 'status',
    title: '状态',
    width: 100,
    slots: { default: 'status' },
  },
  { field: 'createdAt', title: '创建时间', minWidth: 180 },
  {
    field: 'action',
    title: '操作',
    width: 280,
    fixed: 'right',
    slots: { default: 'action' },
  },
];

const [Grid, gridApi] = useVbenVxeGrid(
  createAdminCrudGridOptions<UserItem>({
    tableTitle: '用户管理',
    columns,
    querySchema,
    pageSize: 20,
    query: async ({ page, pageSize, formValues }) => {
      listErrorMessage.value = null;
      try {
        const username =
          typeof formValues.username === 'string'
            ? formValues.username.trim()
            : '';
        const params = {
          ...toPageParams({ page, pageSize }),
          ...(username ? { username } : {}),
        };
        const data = await queryClient.fetchQuery({
          queryKey: qk.system.users(params),
          queryFn: () => getUserListApi(params, withListDetailErrorMode()),
        });
        return toVxeProxyResult(data);
      } catch (error) {
        listErrorMessage.value = apiErrorMessage(error, '加载失败');
        return { items: [], total: 0 };
      }
    },
  }),
);

async function invalidateUsers() {
  await invalidateListDetailQueries(queryClient, ['system', 'users']);
}

const form = useAdminCrudForm<UserFormValues>({
  createDefaults: () => ({
    username: '',
    password: '',
    roles: ['viewer'],
  }),
  mapRecordToValues: (record) => {
    const row = record as UserItem;
    return {
      username: row.username,
      password: '',
      roles: Array.isArray(row.roles) ? [...row.roles] : [],
    };
  },
  submit: async ({ mode, values, record }) => {
    if (mode === 'create') {
      await createUserApi({
        username: values.username,
        password: values.password,
        roles: values.roles,
      });
      message.success('新增成功');
      await invalidateUsers();
      return;
    }
    const row = record as UserItem;
    const payload: UserManageApi.UpdateUserParams = {
      roles: values.roles,
    };
    if (values.password) {
      payload.password = values.password;
    }
    await updateUserApi(row.id, payload);
    message.success('编辑成功');
    await invalidateUsers();
  },
  errorFallback: '操作失败',
});

function buildFormSchema(mode: 'create' | 'edit'): VbenFormSchema[] {
  const isEdit = mode === 'edit';
  return [
    {
      component: 'Input',
      fieldName: 'username',
      label: '用户名',
      componentProps: {
        disabled: isEdit,
        placeholder: '请输入用户名',
      },
      rules: z.string().min(1, { message: '请输入用户名' }),
    },
    {
      component: 'InputPassword',
      fieldName: 'password',
      label: isEdit ? '新密码（留空则不修改）' : '密码',
      componentProps: {
        placeholder: '请输入密码',
      },
      rules: isEdit
        ? z.string().optional()
        : z.string().min(1, { message: '请输入密码' }),
    },
    {
      component: 'Select',
      fieldName: 'roles',
      label: '角色',
      componentProps: {
        mode: 'multiple',
        options: roleOptions.value,
        placeholder: '请选择角色',
        class: 'w-full',
      },
    },
  ];
}

const [Form, formApi] = useVbenForm({
  commonConfig: {
    componentProps: {
      class: 'w-full',
    },
  },
  layout: 'vertical',
  schema: buildFormSchema('create'),
  showDefaultActions: false,
  handleSubmit: async (values) => {
    const ok = await form.handleSubmit(values as UserFormValues);
    if (ok) {
      await gridApi.reload();
    }
  },
});

async function syncFormOpen() {
  formApi.setState({ schema: buildFormSchema(form.mode.value) });
  await nextTick();
  await formApi.setValues(form.initialValues.value as UserFormValues);
}

watch(
  () => form.open.value,
  async (open) => {
    if (open) {
      await syncFormOpen();
    }
  },
);

// 角色选项异步加载后，刷新已打开表单中的 Select options
watch(roleOptions, () => {
  if (form.open.value) {
    formApi.updateSchema([
      {
        fieldName: 'roles',
        componentProps: {
          mode: 'multiple',
          options: roleOptions.value,
          placeholder: '请选择角色',
          class: 'w-full',
        },
      },
    ]);
  }
});

function handleAdd() {
  form.openCreate();
}

function handleEdit(record: UserItem) {
  form.openEdit(record);
}

function handleModalOk() {
  void formApi.validateAndSubmitForm();
}

function handleModalCancel() {
  form.close();
}

async function handleDelete(id: number) {
  try {
    await deleteUserApi(id);
    message.success('删除成功');
    await invalidateUsers();
    await gridApi.reload();
  } catch (error) {
    message.error(apiErrorMessage(error, '删除失败'));
  }
}

async function handleToggleStatus(record: UserItem) {
  try {
    await toggleUserStatusApi(record.id);
    message.success(record.status === 1 ? '用户已冻结' : '用户已解冻');
    await invalidateUsers();
    await gridApi.reload();
  } catch (error) {
    message.error(apiErrorMessage(error, '操作失败'));
  }
}

function roleLabel(role: string) {
  return ROLE_LABEL[role] || role;
}

async function fetchRoleOptions() {
  try {
    const list = await queryClient.fetchQuery({
      queryKey: qk.system.roles({ for: 'user-form' }),
      queryFn: () => getRoleListApi(withListDetailErrorMode()),
    });
    if (Array.isArray(list)) {
      roleOptions.value = list.map((role) => ({
        label: role.displayName || role.name,
        value: role.name,
      }));
    }
  } catch {
    // 选项为空时表单仍可打开；错误已 suppress，不双 toast
  }
}

onMounted(() => {
  void fetchRoleOptions();
});
</script>

<template>
  <div class="p-5">
    <Alert
      v-if="listErrorMessage"
      type="error"
      show-icon
      class="mb-3"
      :message="listErrorMessage"
      closable
      @close="listErrorMessage = null"
    />

    <Grid>
      <template #toolbar-tools>
        <Button type="primary" class="mr-2" @click="handleAdd">
          新增用户
        </Button>
      </template>

      <template #roles="{ row }">
        <Space>
          <Tag
            v-for="role in row.roles"
            :key="role"
            :color="role === 'admin' ? 'green' : 'blue'"
          >
            {{ roleLabel(role) }}
          </Tag>
        </Space>
      </template>

      <template #status="{ row }">
        <Tag :color="row.status === 1 ? 'green' : 'red'">
          {{ row.status === 1 ? '启用' : '禁用' }}
        </Tag>
      </template>

      <template #action="{ row }">
        <UserActionCell
          :row="row as UserItem"
          @edit="handleEdit(row as UserItem)"
          @toggle-status="handleToggleStatus(row as UserItem)"
          @delete="handleDelete"
        />
      </template>
    </Grid>

    <Modal
      :open="form.open.value"
      :title="form.mode.value === 'create' ? '新增用户' : '编辑用户'"
      :confirm-loading="form.submitLoading.value"
      destroy-on-hidden
      @cancel="handleModalCancel"
      @ok="handleModalOk"
    >
      <Alert
        v-if="form.errorMessage.value"
        type="error"
        show-icon
        class="mb-3"
        :message="form.errorMessage.value"
      />
      <div class="pt-2">
        <Form />
      </div>
    </Modal>
  </div>
</template>
