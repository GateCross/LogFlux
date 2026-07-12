<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import {
  Alert,
  Button,
  message,
  Modal,
  Popconfirm,
  Space,
  Tag,
} from 'antdv-next';

import { useVbenForm } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  createMenuApi,
  deleteMenuApi,
  getMenuListApi,
  updateMenuApi,
} from '#/api/system/menu';
import { getRoleListApi } from '#/api/system/role';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { apiErrorMessage } from '#/utils/api-error-message';

import {
  createAdminCrudGridOptions,
  useAdminCrudForm,
} from '../_pattern';
import MenuTitleCell from './components/MenuTitleCell.vue';
import { menuGridColumns } from './menu-columns';
import { buildMenuFormSchema } from './menu-form-schema';
import type { MenuFormValues, MenuViewItem } from './menu-utils';
import {
  buildMenuPayload,
  childCreateOverrides,
  createMenuFormDefaults,
  flattenTree,
  mapMenuRecordToFormValues,
  normalizeMenuItem,
} from './menu-utils';

defineOptions({ name: 'ManageMenu' });

const queryClient = useQueryClient();
const listErrorMessage = ref<string | null>(null);
const roleOptions = ref<Array<{ label: string; value: string }>>([]);
/** 扁平菜单，供上级菜单 Select 与「新增子级」使用 */
const flatList = ref<MenuViewItem[]>([]);
/** 打开「新增子级」时覆盖 create 默认值 */
const createOverrides = ref<Partial<MenuFormValues> | null>(null);

const [Grid, gridApi] = useVbenVxeGrid(
  createAdminCrudGridOptions<MenuViewItem>({
    tableTitle: '菜单管理',
    columns: menuGridColumns,
    pageSize: 100,
    query: async () => {
      listErrorMessage.value = null;
      try {
        const raw = await queryClient.fetchQuery({
          queryKey: qk.system.menus(),
          queryFn: () => getMenuListApi(withListDetailErrorMode()),
        });
        const tree = (Array.isArray(raw) ? raw : []).map(normalizeMenuItem);
        const flattened = flattenTree(tree, []);
        flatList.value = flattened;
        // 二级菜单默认折叠
        return { items: tree, total: flattened.length };
      } catch (error) {
        flatList.value = [];
        listErrorMessage.value = apiErrorMessage(error, '加载菜单列表失败');
        return { items: [], total: 0 };
      }
    },
    gridOptions: {
      // 菜单为树形全量列表，无服务端分页
      pagerConfig: { enabled: false },
      treeConfig: {
        childrenField: 'children',
        rowField: 'id',
        expandAll: false,
      },
      scrollX: { enabled: true },
    },
  }),
);

async function invalidateMenus() {
  await invalidateListDetailQueries(queryClient, qk.system.menus());
}

const form = useAdminCrudForm<MenuFormValues>({
  createDefaults: () => {
    const base = createMenuFormDefaults(0);
    return createOverrides.value
      ? { ...base, ...createOverrides.value }
      : base;
  },
  mapRecordToValues: (record) =>
    mapMenuRecordToFormValues(record as MenuViewItem),
  submit: async ({ mode, values, record }) => {
    const payload = buildMenuPayload(values);
    if (mode === 'create') {
      await createMenuApi(payload);
    } else {
      await updateMenuApi((record as MenuViewItem).id, payload);
    }
    message.success('操作成功');
    await invalidateMenus();
  },
  errorFallback: '操作失败',
});

function currentFormSchema() {
  return buildMenuFormSchema({
    mode: form.mode.value,
    flatList: flatList.value,
    editingId:
      form.mode.value === 'edit'
        ? ((form.editingRecord.value as MenuViewItem | null)?.id ?? null)
        : null,
    roleOptions: roleOptions.value,
  });
}

const [Form, formApi] = useVbenForm({
  commonConfig: {
    componentProps: { class: 'w-full' },
  },
  layout: 'vertical',
  schema: buildMenuFormSchema({
    mode: 'create',
    flatList: [],
    editingId: null,
    roleOptions: [],
  }),
  showDefaultActions: false,
  handleSubmit: async (values) => {
    const normalized: MenuFormValues = {
      ...(values as MenuFormValues),
      parentId:
        values.parentId === undefined || values.parentId === null
          ? 0
          : (values.parentId as number),
    };
    const ok = await form.handleSubmit(normalized);
    if (ok) {
      createOverrides.value = null;
      await gridApi.reload();
    }
  },
});

async function syncFormOpen() {
  formApi.setState({ schema: currentFormSchema() });
  await nextTick();
  const values = { ...(form.initialValues.value as MenuFormValues) };
  // Select 展示：0 视为空（一级菜单）
  await formApi.setValues({
    ...values,
    parentId: values.parentId || undefined,
  } as MenuFormValues);
}

watch(
  () => form.open.value,
  async (open) => {
    if (open) {
      await syncFormOpen();
    } else {
      createOverrides.value = null;
    }
  },
);

watch(roleOptions, () => {
  if (!form.open.value) return;
  formApi.updateSchema([
    {
      fieldName: 'roles',
      componentProps: {
        mode: 'multiple',
        options: roleOptions.value,
        placeholder: '留空表示公开',
        class: 'w-full',
      },
    },
  ]);
});

function handleAddRoot() {
  createOverrides.value = null;
  form.openCreate();
}

function handleAddChild(record: MenuViewItem) {
  createOverrides.value = childCreateOverrides(record);
  form.openCreate();
}

function handleEdit(record: MenuViewItem) {
  createOverrides.value = null;
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
    await deleteMenuApi(id);
    message.success('删除成功');
    await invalidateMenus();
    await gridApi.reload();
  } catch (error) {
    message.error(apiErrorMessage(error, '删除菜单失败'));
  }
}

async function fetchRoleOptions() {
  try {
    const list = await queryClient.fetchQuery({
      queryKey: qk.system.roles({ for: 'menu-form' }),
      queryFn: () => getRoleListApi(withListDetailErrorMode()),
    });
    if (Array.isArray(list)) {
      roleOptions.value = list.map((role) => ({
        label: role.displayName || role.name,
        value: role.name,
      }));
    }
  } catch {
    // 选项为空时表单仍可打开
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
        <Button type="primary" class="mr-2" @click="handleAddRoot">
          新增一级菜单
        </Button>
      </template>

      <template #title="{ row }">
        <MenuTitleCell
          :icon="row.icon"
          :local-icon="row.localIcon"
          :title="row.displayTitle"
        />
      </template>

      <template #roles="{ row }">
        <template v-if="row.roles?.length">
          <Tag v-for="role in row.roles" :key="role" color="blue">
            {{ role }}
          </Tag>
        </template>
        <Tag v-else>公开</Tag>
      </template>

      <template #hideInMenu="{ row }">
        <Tag :color="row.hideInMenu ? 'default' : 'green'">
          {{ row.hideInMenu ? '隐藏' : '显示' }}
        </Tag>
      </template>

      <template #action="{ row }">
        <Space :size="6">
          <Button
            size="small"
            class="table-action-btn table-action-btn--secondary"
            @click="handleAddChild(row as MenuViewItem)"
          >
            新增子级
          </Button>
          <Button
            size="small"
            class="table-action-btn"
            @click="handleEdit(row as MenuViewItem)"
          >
            编辑
          </Button>
          <Popconfirm
            title="确认删除该菜单及其所有子项？"
            ok-text="确认"
            cancel-text="取消"
            @confirm="handleDelete(row.id)"
          >
            <Button
              size="small"
              danger
              class="table-action-btn table-action-btn--danger"
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      </template>
    </Grid>

    <Modal
      :open="form.open.value"
      :title="form.mode.value === 'create' ? '新增菜单' : '编辑菜单'"
      :confirm-loading="form.submitLoading.value"
      :width="680"
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
