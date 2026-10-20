<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { VxeTableGridColumns } from '#/adapter/vxe-table';

import type { UserManageApi } from '#/api/system/user';

import { nextTick, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import { Alert, Button, message, Modal, Tag } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  createUserApi,
  deleteUserApi,
  getUserListApi,
  toggleUserStatusApi,
  updateUserApi,
} from '#/api/system/user';
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

defineOptions({ name: 'SystemUser' });

type UserItem = UserManageApi.UserItem;

interface UserFormValues extends Record<string, unknown> {
  password: string;
  username: string;
}

const queryClient = useQueryClient();
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
    field: 'status',
    title: '状态',
    width: 100,
    slots: { default: 'status' },
  },
  { field: 'createdAt', title: '创建时间', minWidth: 180 },
  {
    field: 'action',
    title: '操作',
    width: 220,
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
  }),
  mapRecordToValues: (record) => {
    const row = record as UserItem;
    return {
      username: row.username,
      password: '',
    };
  },
  submit: async ({ mode, values, record }) => {
    if (mode === 'create') {
      await createUserApi({
        username: values.username,
        password: values.password,
      });
      message.success('新增成功');
      await invalidateUsers();
      return;
    }
    const row = record as UserItem;
    await updateUserApi(row.id, {
      ...(values.password ? { password: values.password } : {}),
    });
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
