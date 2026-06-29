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
  Table,
  Tag,
  Tooltip,
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
const hiddenPermissionValues = new Set(['caddy_source', 'user_center']);

interface PermissionOption {
  label: string;
  value: string;
}

interface PermissionGroup {
  label: string;
  options: PermissionOption[];
}

interface PermissionSummary {
  details: string[];
  group: string;
  label: string;
}

const permissionGroups: PermissionGroup[] = [
  {
    label: '仪表盘',
    options: [{ label: '仪表盘', value: 'dashboard' }],
  },
  {
    label: 'Caddy 配置',
    options: [
      { label: 'Caddy 菜单', value: 'caddy' },
      { label: '配置管理', value: 'caddy_config' },
      { label: '访问控制', value: 'caddy_access' },
      { label: '访问日志', value: 'logs_caddy' },
    ],
  },
  {
    label: '日志与审计',
    options: [{ label: '系统日志', value: 'logs' }],
  },
  {
    label: '定时任务',
    options: [{ label: '定时任务', value: 'cron' }],
  },
  {
    label: '系统管理',
    options: [
      { label: '系统管理', value: 'manage' },
      { label: '用户管理', value: 'manage_user' },
      { label: '角色管理', value: 'manage_role' },
      { label: '菜单管理', value: 'manage_menu' },
    ],
  },
  {
    label: '通知管理',
    options: [
      { label: '通知管理', value: 'notification' },
      { label: '通知渠道', value: 'notification_channel' },
      { label: '通知规则', value: 'notification_rule' },
      { label: '通知模板', value: 'notification_template' },
      { label: '发送日志', value: 'notification_log' },
    ],
  },
];

const permissionOptionMap = new Map(
  permissionGroups.flatMap((group) =>
    group.options.map((option) => [option.value, { ...option, group: group.label }] as const),
  ),
);

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

const visiblePermissionGroups = computed<PermissionGroup[]>(() => {
  const extraOptions = uniquePermissions(currentPermissions.value)
    .filter((permission) => !permissionOptionMap.has(permission))
    .map((permission) => ({
      label: permission,
      value: permission,
    }));

  if (extraOptions.length === 0) {
    return permissionGroups;
  }

  return [
    ...permissionGroups,
    {
      label: '其他权限',
      options: extraOptions,
    },
  ];
});

function uniquePermissions(permissions?: string[]) {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const permission of permissions ?? []) {
    const value = String(permission || '').trim();
    if (!value || seen.has(value) || hiddenPermissionValues.has(value)) continue;
    seen.add(value);
    result.push(value);
  }
  return result;
}

function permissionLabelOf(permission: string) {
  return permissionOptionMap.get(permission)?.label || permission;
}

function buildPermissionSummaries(permissions?: string[]): PermissionSummary[] {
  const selected = new Set(uniquePermissions(permissions));
  const summaries: PermissionSummary[] = [];

  for (const group of permissionGroups) {
    const details = group.options
      .filter((option) => selected.has(option.value))
      .map((option) => option.label);
    if (details.length === 0) continue;

    summaries.push({
      details,
      group: group.label,
      label: details.length === 1 ? details[0]! : `${group.label} ${details.length}项`,
    });
  }

  const extraDetails = [...selected]
    .filter((permission) => !permissionOptionMap.has(permission))
    .map(permissionLabelOf);

  if (extraDetails.length > 0) {
    summaries.push({
      details: extraDetails,
      group: '其他权限',
      label: extraDetails.length === 1 ? extraDetails[0]! : `其他权限 ${extraDetails.length}项`,
    });
  }

  return summaries;
}

function visiblePermissionSummaries(permissions?: string[]) {
  return buildPermissionSummaries(permissions).slice(0, 3);
}

function hiddenPermissionSummaries(permissions?: string[]) {
  return buildPermissionSummaries(permissions).slice(3);
}

function hasVisiblePermissions(permissions?: string[]) {
  return uniquePermissions(permissions).length > 0;
}

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
  currentPermissions.value = uniquePermissions(record.permissions);
  modalVisible.value = true;
}

async function handleSubmit() {
  if (!currentRoleId.value) return;

  submitting.value = true;
  try {
    await updateRolePermissionsApi(currentRoleId.value, uniquePermissions(currentPermissions.value));
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
            <div v-if="hasVisiblePermissions(record.permissions)" class="permission-summary">
              <Tooltip
                v-for="summary in visiblePermissionSummaries(record.permissions)"
                :key="summary.group"
              >
                <template #title>{{ summary.details.join('、') }}</template>
                <Tag color="blue">{{ summary.label }}</Tag>
              </Tooltip>
              <Tooltip v-if="hiddenPermissionSummaries(record.permissions).length">
                <template #title>
                  <div class="permission-tooltip-list">
                    <div
                      v-for="summary in hiddenPermissionSummaries(record.permissions)"
                      :key="summary.group"
                    >
                      {{ summary.label }}：{{ summary.details.join('、') }}
                    </div>
                  </div>
                </template>
                <Tag class="permission-more-tag">+{{ hiddenPermissionSummaries(record.permissions).length }}</Tag>
              </Tooltip>
            </div>
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
          <CheckboxGroup v-model:value="currentPermissions" class="permission-checkbox-group">
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
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>

<style scoped>
.permission-summary {
  display: flex;
  max-width: 280px;
  flex-wrap: wrap;
  gap: 6px;
}

.permission-summary :deep(.ant-tag) {
  margin-inline-end: 0;
}

.permission-more-tag {
  cursor: help;
}

.permission-checkbox-group {
  width: 100%;
}

.permission-groups {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px 24px;
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
