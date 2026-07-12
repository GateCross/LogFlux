<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { VxeTableGridColumns } from '#/adapter/vxe-table';

import type { RoleApi } from '#/api/system/role';

import { computed, nextTick, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import {
  Alert,
  Button,
  Checkbox,
  CheckboxGroup,
  message,
  Modal,
  Space,
  Tag,
  Tooltip,
} from 'antdv-next';

import { useVbenForm } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  getRoleListApi,
  updateRolePermissionsApi,
} from '#/api/system/role';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { apiErrorMessage } from '#/utils/api-error-message';

import {
  createAdminCrudGridOptions,
  useAdminCrudForm,
} from '../_pattern';
import {
  buildVisiblePermissionGroups,
  hasVisiblePermissions,
  hiddenPermissionSummaries,
  uniquePermissions,
  visiblePermissionSummaries,
} from './permission-utils';

defineOptions({ name: 'ManageRole' });

type RoleItem = RoleApi.RoleItem;

interface RolePermissionFormValues extends Record<string, unknown> {
  permissions: string[];
}

const queryClient = useQueryClient();
const listErrorMessage = ref<string | null>(null);

const columns: VxeTableGridColumns<RoleItem> = [
  { field: 'id', title: 'ID', width: 80 },
  { field: 'name', title: '角色名', width: 140 },
  { field: 'displayName', title: '显示名称', width: 160 },
  { field: 'description', title: '描述', minWidth: 120 },
  {
    field: 'permissions',
    title: '权限',
    align: 'left',
    headerAlign: 'center',
    showOverflow: false,
    width: 280,
    slots: { default: 'permissions' },
  },
  { field: 'createdAt', title: '创建时间', width: 180 },
  {
    field: 'action',
    title: '操作',
    width: 180,
    fixed: 'right',
    slots: { default: 'action' },
  },
];

const [Grid, gridApi] = useVbenVxeGrid(
  createAdminCrudGridOptions<RoleItem>({
    tableTitle: '角色管理',
    columns,
    pageSize: 100,
    query: async () => {
      listErrorMessage.value = null;
      try {
        const list = await queryClient.fetchQuery({
          queryKey: qk.system.roles(),
          queryFn: () => getRoleListApi(withListDetailErrorMode()),
        });
        const items = Array.isArray(list) ? list : [];
        return { items, total: items.length };
      } catch (error) {
        listErrorMessage.value = apiErrorMessage(error, '加载角色列表失败');
        return { items: [], total: 0 };
      }
    },
    gridOptions: {
      // 角色列表为全量接口，无服务端分页
      pagerConfig: {
        enabled: false,
      },
    },
  }),
);

const form = useAdminCrudForm<RolePermissionFormValues>({
  createDefaults: () => ({
    permissions: [],
  }),
  mapRecordToValues: (record) => {
    const row = record as RoleItem;
    return {
      permissions: uniquePermissions(row.permissions),
    };
  },
  submit: async ({ values, record }) => {
    const row = record as RoleItem | null;
    if (!row?.id) {
      throw new Error('未选择角色');
    }
    await updateRolePermissionsApi(
      row.id,
      uniquePermissions(values.permissions),
    );
    message.success('权限更新成功');
    await invalidateListDetailQueries(queryClient, ['system', 'roles']);
  },
  errorFallback: '操作失败',
});

const modalTitle = computed(() => {
  const row = form.editingRecord.value as RoleItem | null;
  if (!row) return '编辑权限';
  const name = row.displayName || row.name;
  return name ? `编辑权限 - ${name}` : '编辑权限';
});

/** 打开弹层时的权限值驱动「其他权限」分组；勾选过程不改分组结构 */
const visiblePermissionGroups = computed(() =>
  buildVisiblePermissionGroups(
    (form.initialValues.value as RolePermissionFormValues).permissions ?? [],
  ),
);

const formSchema: VbenFormSchema[] = [
  {
    component: 'CheckboxGroup',
    fieldName: 'permissions',
    label: '权限列表',
    formItemClass: 'items-start',
    componentProps: {
      class: 'permission-checkbox-group w-full',
    },
  },
];

const [Form, formApi] = useVbenForm({
  commonConfig: {
    componentProps: {
      class: 'w-full',
    },
  },
  layout: 'vertical',
  schema: formSchema,
  showDefaultActions: false,
  handleSubmit: async (values) => {
    const ok = await form.handleSubmit(values as RolePermissionFormValues);
    if (ok) {
      await gridApi.reload();
    }
  },
});

async function syncFormOpen() {
  await nextTick();
  await formApi.setValues(
    form.initialValues.value as RolePermissionFormValues,
  );
}

watch(
  () => form.open.value,
  async (open) => {
    if (open) {
      await syncFormOpen();
    }
  },
);

function handleEditPermissions(record: RoleItem) {
  form.openEdit(record);
}

function handleModalOk() {
  void formApi.validateAndSubmitForm();
}

function handleModalCancel() {
  form.close();
}

function slotPermissionValue(slotProps: Record<string, any>): string[] {
  const raw = slotProps?.value ?? slotProps?.modelValue;
  return Array.isArray(raw) ? raw : [];
}

function setSlotPermissionValue(
  slotProps: Record<string, any>,
  next: string[],
) {
  const updater =
    slotProps?.['onUpdate:value'] ?? slotProps?.['onUpdate:modelValue'];
  updater?.(next);
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
      <template #permissions="{ row }">
        <div
          v-if="hasVisiblePermissions(row.permissions)"
          class="permission-summary"
        >
          <Tooltip
            v-for="summary in visiblePermissionSummaries(row.permissions)"
            :key="summary.group"
          >
            <template #title>{{ summary.details.join('、') }}</template>
            <Tag color="blue">{{ summary.label }}</Tag>
          </Tooltip>
          <Tooltip v-if="hiddenPermissionSummaries(row.permissions).length">
            <template #title>
              <div class="permission-tooltip-list">
                <div
                  v-for="summary in hiddenPermissionSummaries(row.permissions)"
                  :key="summary.group"
                >
                  {{ summary.label }}：{{ summary.details.join('、') }}
                </div>
              </div>
            </template>
            <Tag class="permission-more-tag">
              +{{ hiddenPermissionSummaries(row.permissions).length }}
            </Tag>
          </Tooltip>
        </div>
        <Tag v-else>无</Tag>
      </template>

      <template #action="{ row }">
        <Space :size="6">
          <Button
            size="small"
            class="table-action-btn"
            @click="handleEditPermissions(row as RoleItem)"
          >
            编辑权限
          </Button>
        </Space>
      </template>
    </Grid>

    <Modal
      :open="form.open.value"
      :title="modalTitle"
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
        <Form>
          <template #permissions="slotProps">
            <CheckboxGroup
              class="permission-checkbox-group w-full"
              :value="slotPermissionValue(slotProps)"
              @update:value="
                (v: string[]) => setSlotPermissionValue(slotProps, v)
              "
            >
              <div class="permission-groups">
                <div
                  v-for="group in visiblePermissionGroups"
                  :key="group.label"
                  class="permission-group"
                >
                  <div class="permission-group-title">
                    {{ group.label }}
                  </div>
                  <div class="permission-options">
                    <Checkbox
                      v-for="option in group.options"
                      :key="option.value"
                      :value="option.value"
                    >
                      {{ option.label }}
                    </Checkbox>
                  </div>
                </div>
              </div>
            </CheckboxGroup>
          </template>
        </Form>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.permission-summary {
  display: flex;
  width: 100%;
  flex-wrap: wrap;
  justify-content: flex-start;
  gap: 6px;
}

.permission-summary :deep(.ant-tag) {
  margin-inline-end: 0;
}

.permission-more-tag {
  cursor: help;
}

.permission-groups {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px 24px;
  width: 100%;
}

.permission-group-title {
  margin-bottom: 8px;
  color: #344054;
  font-size: 14px;
  font-weight: 600;
}

.permission-options {
  display: grid;
  gap: 8px;
}

.permission-tooltip-list {
  line-height: 1.8;
}

@media (max-width: 640px) {
  .permission-groups {
    grid-template-columns: 1fr;
  }
}
</style>
