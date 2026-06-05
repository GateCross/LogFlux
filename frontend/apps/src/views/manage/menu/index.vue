<script lang="ts" setup>
import type { MenuApi } from '#/api/system/menu';

import { computed, onMounted, reactive, ref } from 'vue';

import { Icon } from '@iconify/vue';

import {
  Button,
  Card,
  Form,
  FormItem,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Select,
  SelectOption,
  Space,
  Switch,
  Table,
  Tag,
} from 'ant-design-vue';

import {
  createMenuApi,
  deleteMenuApi,
  getMenuListApi,
  updateMenuApi,
} from '#/api/system/menu';
import { findRouteTitle } from '#/router/route-title';
import { getRoleListApi } from '#/api/system/role';

defineOptions({ name: 'ManageMenu' });

interface MenuItem extends MenuApi.MenuItem {
  children?: MenuItem[];
  displayTitle: string;
  hideInMenu: boolean;
  i18nKey: string;
  icon: string;
  localIcon: string;
  roles: string[];
  title: string;
  type: 'directory' | 'menu';
}

const loading = ref(false);
const dataSource = ref<MenuItem[]>([]);
const flatList = ref<MenuItem[]>([]);
const roleOptions = ref<Array<{ label: string; value: string }>>([]);

const modalVisible = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const editingId = ref<number | null>(null);
const submitting = ref(false);

const modalTitle = computed(() =>
  modalType.value === 'add' ? '新增菜单' : '编辑菜单',
);

const formState = reactive({
  component: '',
  hideInMenu: false,
  i18nKey: '',
  icon: '',
  localIcon: '',
  name: '',
  order: 0,
  parentId: 0 as number | null,
  path: '',
  roles: [] as string[],
  title: '',
});

const columns = [
  { dataIndex: 'displayTitle', key: 'title', title: '菜单名称', width: 260 },
  { dataIndex: 'path', key: 'path', title: '路径', ellipsis: true, width: 180 },
  { dataIndex: 'component', key: 'component', title: '组件', ellipsis: true, width: 220 },
  { dataIndex: 'order', key: 'order', title: '排序', width: 80 },
  { dataIndex: 'i18nKey', key: 'i18nKey', title: '国际化 Key', ellipsis: true, width: 180 },
  { dataIndex: 'roles', key: 'roles', title: '所需角色', width: 220 },
  { dataIndex: 'hideInMenu', key: 'hideInMenu', title: '菜单显示', width: 100 },
  { key: 'actions', title: '操作', width: 220 },
];

const parentOptions = computed(() =>
  flatList.value.filter((item) => item.id !== editingId.value),
);

function resolveMenuTitle(item: MenuApi.MenuItem, rawTitle: string) {
  return findRouteTitle(item.name, rawTitle, item.meta?.i18nKey) || rawTitle || item.name;
}

function normalizeMenuItem(item: MenuApi.MenuItem): MenuItem {
  const meta = item.meta ?? {};
  const children = item.children?.map(normalizeMenuItem) ?? [];
  const rawTitle = meta.title ?? item.name;
  return {
    ...item,
    children: children.length > 0 ? children : undefined,
    displayTitle: resolveMenuTitle(item, rawTitle),
    hideInMenu: Boolean(meta.hideInMenu),
    i18nKey: meta.i18nKey ?? '',
    icon: meta.icon ?? '',
    localIcon: meta.localIcon ?? '',
    meta,
    parentId: item.parentId ?? 0,
    requiredRoles: item.requiredRoles ?? [],
    roles: item.requiredRoles?.length ? item.requiredRoles : meta.roles ?? [],
    title: rawTitle,
    type: children.length > 0 || item.component.startsWith('layout.')
      ? 'directory'
      : 'menu',
  };
}

function flattenTree(nodes: MenuItem[], result: MenuItem[] = []): MenuItem[] {
  for (const node of nodes) {
    result.push(node);
    if (node.children?.length) {
      flattenTree(node.children, result);
    }
  }
  return result;
}

function resetForm(parentId = 0) {
  Object.assign(formState, {
    component: parentId > 0 ? 'view.' : 'layout.base',
    hideInMenu: false,
    i18nKey: '',
    icon: '',
    localIcon: '',
    name: '',
    order: 0,
    parentId,
    path: parentId > 0 ? '/' : '',
    roles: [],
    title: '',
  });
}

function buildPayload(): MenuApi.MenuPayload {
  const title = formState.title || formState.name;
  return {
    component: formState.component,
    meta: {
      hideInMenu: formState.hideInMenu,
      i18nKey: formState.i18nKey,
      icon: formState.icon,
      localIcon: formState.localIcon,
      order: formState.order,
      roles: formState.roles,
      title,
    },
    name: formState.name,
    order: formState.order,
    parentId: formState.parentId ?? 0,
    path: formState.path,
    requiredRoles: formState.roles,
  };
}

async function fetchRoles() {
  const roles = await getRoleListApi();
  roleOptions.value = roles.map((role) => ({
    label: role.displayName || role.name,
    value: role.name,
  }));
}

async function fetchData() {
  loading.value = true;
  try {
    dataSource.value = (await getMenuListApi()).map(normalizeMenuItem);
    flatList.value = flattenTree(dataSource.value, []);
  } catch {
    message.error('加载菜单列表失败');
  } finally {
    loading.value = false;
  }
}

function openAddModal(parentId = 0) {
  modalType.value = 'add';
  editingId.value = null;
  resetForm(parentId);
  modalVisible.value = true;
}

function openAddChild(record: MenuItem) {
  openAddModal(record.id);
  formState.path = `${record.path.replace(/\/$/, '')}/`;
}

function openEditModal(record: MenuItem) {
  modalType.value = 'edit';
  editingId.value = record.id;
  Object.assign(formState, {
    component: record.component,
    hideInMenu: record.hideInMenu,
    i18nKey: record.i18nKey,
    icon: record.icon,
    localIcon: record.localIcon,
    name: record.name,
    order: record.order || record.meta?.order || 0,
    parentId: record.parentId ?? 0,
    path: record.path,
    roles: [...record.roles],
    title: record.title,
  });
  modalVisible.value = true;
}

async function handleSubmit() {
  if (!formState.name.trim()) {
    message.warning('请输入菜单唯一标识');
    return;
  }
  if (!formState.path.trim()) {
    message.warning('请输入菜单路径');
    return;
  }
  if (!formState.component.trim()) {
    message.warning('请输入组件路径');
    return;
  }

  submitting.value = true;
  try {
    if (editingId.value) {
      await updateMenuApi(editingId.value, buildPayload());
    } else {
      await createMenuApi(buildPayload());
    }
    message.success('操作成功');
    modalVisible.value = false;
    await fetchData();
  } catch {
    message.error('操作失败');
  } finally {
    submitting.value = false;
  }
}

async function handleDelete(id: number) {
  try {
    await deleteMenuApi(id);
    message.success('删除成功');
    await fetchData();
  } catch {
    message.error('删除菜单失败');
  }
}

function iconName(record: MenuItem) {
  return record.icon;
}

onMounted(() => {
  void fetchRoles();
  void fetchData();
});
</script>

<template>
  <div class="p-5">
    <Card title="菜单管理">
      <template #extra>
        <Space>
          <Button @click="fetchData">刷新</Button>
          <Button type="primary" @click="openAddModal()">新增一级菜单</Button>
        </Space>
      </template>

      <Table
        :columns="columns"
        :data-source="dataSource"
        :default-expand-all-rows="true"
        :loading="loading"
        :pagination="false"
        row-key="id"
        :scroll="{ x: 1260 }"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'title'">
            <div class="menu-title-cell">
              <Icon v-if="iconName(record as MenuItem)" :icon="iconName(record as MenuItem)" class="menu-icon" />
              <span class="menu-title-text">{{ record.displayTitle }}</span>
              <span v-if="record.localIcon && !record.icon" class="menu-icon-key">{{ record.localIcon }}</span>
            </div>
          </template>

          <template v-if="column.key === 'roles'">
            <template v-if="record.roles?.length">
              <Tag v-for="role in record.roles" :key="role" color="blue">
                {{ role }}
              </Tag>
            </template>
            <Tag v-else>公开</Tag>
          </template>

          <template v-if="column.key === 'hideInMenu'">
            <Tag :color="record.hideInMenu ? 'default' : 'green'">
              {{ record.hideInMenu ? '隐藏' : '显示' }}
            </Tag>
          </template>

          <template v-if="column.key === 'actions'">
            <Space>
              <Button type="link" size="small" @click="openAddChild(record as MenuItem)">
                新增子级
              </Button>
              <Button type="link" size="small" @click="openEditModal(record as MenuItem)">
                编辑
              </Button>
              <Popconfirm
                title="确认删除该菜单及其所有子项？"
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
      :confirm-loading="submitting"
      :open="modalVisible"
      :title="modalTitle"
      :width="680"
      @cancel="modalVisible = false"
      @ok="handleSubmit"
    >
      <Form layout="vertical" class="mt-4">
        <FormItem label="上级菜单">
          <Select
            allow-clear
            :value="formState.parentId || undefined"
            @change="(value: unknown) => (formState.parentId = typeof value === 'number' ? value : 0)"
          >
            <SelectOption
              v-for="item in parentOptions"
              :key="item.id"
              :value="item.id"
            >
              {{ item.displayTitle }} ({{ item.path }})
            </SelectOption>
          </Select>
        </FormItem>

        <FormItem label="菜单唯一标识" required>
          <Input v-model:value="formState.name" placeholder="例如 dashboard" />
        </FormItem>

        <FormItem label="路径" required>
          <Input v-model:value="formState.path" placeholder="例如 /dashboard" />
        </FormItem>

        <FormItem label="组件" required>
          <Input
            v-model:value="formState.component"
            placeholder="例如 layout.base 或 view.dashboard"
          />
        </FormItem>

        <FormItem label="显示名称">
          <Input v-model:value="formState.title" placeholder="默认使用菜单唯一标识" />
        </FormItem>

        <FormItem label="国际化 Key">
          <Input v-model:value="formState.i18nKey" placeholder="例如 route.dashboard" />
        </FormItem>

        <FormItem label="图标">
          <Input v-model:value="formState.icon" placeholder="例如 mdi:home" />
        </FormItem>

        <FormItem label="本地图标">
          <Input v-model:value="formState.localIcon" />
        </FormItem>

        <FormItem label="排序">
          <InputNumber v-model:value="formState.order" class="w-full" />
        </FormItem>

        <FormItem label="所需角色">
          <Select
            v-model:value="formState.roles"
            mode="multiple"
            placeholder="留空表示公开"
          >
            <SelectOption
              v-for="role in roleOptions"
              :key="role.value"
              :value="role.value"
            >
              {{ role.label }}
            </SelectOption>
          </Select>
        </FormItem>

        <FormItem label="隐藏菜单">
          <Switch v-model:checked="formState.hideInMenu" />
        </FormItem>
      </Form>
    </Modal>
  </div>
</template>

<style scoped>
.menu-title-cell {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
}

.menu-icon {
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
  color: #1677ff;
}

.menu-title-text {
  min-width: 0;
  overflow: hidden;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-icon-key {
  max-width: 88px;
  flex: 0 1 auto;
  overflow: hidden;
  padding: 1px 6px;
  color: #1677ff;
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: #e6f4ff;
  border: 1px solid #91caff;
  border-radius: 4px;
}
</style>
