<template>
  <div class="h-full overflow-hidden">
    <n-card :title="$t('route.manage_menu')" :bordered="false" class="h-full rounded-8px shadow-sm">
      <div class="flex flex-col h-full">
        <n-space class="pb-12px" justify="space-between">
          <n-space>
            <n-button type="primary" @click="fetchData">
              <template #icon>
                <icon-ic-round-refresh />
              </template>
              {{ $t('common.refresh') }}
            </n-button>
            <n-button type="primary" ghost @click="handleAddRoot">
              <template #icon>
                <icon-ic-round-plus />
              </template>
              {{ $t('common.add') + '一级菜单' }}
            </n-button>
          </n-space>
        </n-space>
        
        <n-data-table
          flex-height
          :columns="columns"
          :data="tableData"
          :loading="loading"
          :row-key="row => row.id"
          class="flex-1-hidden"
          default-expand-all
          expand-column-key="expander"
        />
      </div>
    </n-card>

    <n-modal v-model:show="showModal" :title="modalTitle" preset="card" class="w-600px">
      <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="100">
        <n-form-item label="上级菜单" path="parentId">
          <n-tree-select
            v-model:value="formModel.parentId"
            :options="menuSelectOptions"
            placeholder="请选择上级菜单（留空为一级菜单）"
            clearable
          />
        </n-form-item>
        <n-form-item label="菜单名称" path="name">
          <n-input v-model:value="formModel.name" placeholder="请输入菜单唯一标识 (e.g. dashboard)" />
        </n-form-item>
        <n-form-item label="路径" path="path">
          <n-input v-model:value="formModel.path" placeholder="请输入路径 (e.g. /dashboard)" />
        </n-form-item>
        <n-form-item label="组件" path="component">
          <n-input v-model:value="formModel.component" placeholder="请输入组件路径 (e.g. view.dashboard)" />
        </n-form-item>
        <n-form-item label="排序" path="order">
          <n-input-number v-model:value="formModel.order" class="w-full" />
        </n-form-item>
        <n-form-item label="I18nKey" path="meta.i18nKey">
          <n-input v-model:value="formModel.meta.i18nKey" placeholder="请输入国际化Key (e.g. route.dashboard)" />
        </n-form-item>
        <n-form-item label="图标" path="meta.icon">
          <n-input v-model:value="formModel.meta.icon" placeholder="请输入图标Key (e.g. mdi:home)" />
        </n-form-item>
        <n-form-item label="本地图标" path="meta.localIcon">
          <n-input v-model:value="formModel.meta.localIcon" placeholder="请输入本地图标名称" />
        </n-form-item>
        <n-form-item label="隐藏菜单" path="meta.hideInMenu">
          <n-switch v-model:value="formModel.meta.hideInMenu" />
        </n-form-item>
        <n-form-item label="所需角色" path="roles">
          <n-select v-model:value="formModel.roles" multiple :options="roleOptions" placeholder="请选择可见角色（留空为公开）" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="closeModal">{{ $t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="submitLoading" @click="handleSubmit">{{ $t('common.confirm') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h, computed } from 'vue';
import { NButton, NPopconfirm, NTag, NIcon, useMessage } from 'naive-ui';
import type { DataTableColumns, FormRules } from 'naive-ui';
import { request } from '@/service/request';
import { $t } from '@/locales';
import { useSvgIcon } from '@/hooks/common/icon';

interface MenuItem {
  id: number;
  name: string;
  path: string;
  component: string;
  order: number;
  meta: {
    title?: string;
    i18nKey?: App.I18n.I18nKey | null;
    icon?: string;
    localIcon?: string;
    order?: number;
    hideInMenu?: boolean;
    roles?: string[];
  };
  roles?: string[];
  requiredRoles?: string[];
  parentId?: number; // Added field
  children?: MenuItem[];
  createdAt: string;
}

const message = useMessage();
const { SvgIconVNode } = useSvgIcon();
const loading = ref(false);
const tableData = ref<MenuItem[]>([]);
const roleOptions = ref<any[]>([]);

const columns: DataTableColumns<MenuItem> = [
  {
    type: 'selection'
  },
  {
    title: $t('common.index'),
    key: 'index',
    width: 60,
    align: 'center',
    render: (row, index) => {
        if (row.parentId && row.parentId > 0) {
            return '';
        }
        return index + 1;
    }
  },
  {
    title: '',
    key: 'expander',
    width: 40,
    render: () => null // Naive UI will handle the expand icon here if expand-column-key is set
  },
  {
    title: '菜单名称',
    key: 'name',
    width: 200,
    render(row) {
      const { icon, localIcon, i18nKey, title } = row.meta;
      const label = i18nKey ? $t(i18nKey) : title || row.name;
      const iconVNode = SvgIconVNode({ icon, localIcon, fontSize: 18 });
      return h('div', { class: 'flex items-center gap-2' }, [
        iconVNode?.(),
        h('span', null, label)
      ]);
    }
  },
  { title: '路径', key: 'path', width: 180 },
  {
    title: '排序',
    key: 'order',
    width: 80,
    align: 'center',
    render(row) {
      // Use meta.order if available (common pattern in this project), fallback to row.order
      return row.order || row.meta?.order || 0;
    }
  },
  {
    title: '国际化Key',
    key: 'i18nKey',
    width: 200,
    render: row => row.meta.i18nKey || '-'
  },
  {
    title: '所需角色',
    key: 'roles',
    render(row) {
      const roles = row.requiredRoles && row.requiredRoles.length > 0 ? row.requiredRoles : (row.meta?.roles || []);
      if (!roles || roles.length === 0) return h(NTag, { size: 'small', bordered: false }, { default: () => '公开' });
      return roles.map(role => h(NTag, { size: 'small', type: 'info', bordered: false, class: 'mr-1' }, { default: () => role }));
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    render(row) {
      return h('div', { class: 'flex gap-2' }, [
        h(NButton, { size: 'small', type: 'primary', ghost: true, onClick: () => handleAddChild(row) }, { default: () => '新增子级' }),
        h(NButton, { size: 'small', type: 'primary', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
        h(
          NPopconfirm,
          { onPositiveClick: () => handleDelete(row.id) },
          {
            default: () => '确认删除该菜单及其所有子项吗？',
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' })
          }
        )
      ]);
    }
  }
];

// Modal
const showModal = ref(false);
const modalType = ref<'add' | 'edit'>('add');
const modalTitle = computed(() => (modalType.value === 'add' ? '新增菜单' : '编辑菜单'));
const submitLoading = ref(false);
const formRef = ref<any>(null);
const formModel = reactive({
  id: 0,
  parentId: null as number | null,
  name: '',
  path: '',
  component: '',
  order: 0,
  meta: {
    title: '',
    i18nKey: '' as App.I18n.I18nKey,
    icon: '',
    localIcon: '',
    hideInMenu: false
  },
  roles: [] as string[]
});

const rules: FormRules = {
  name: { required: true, message: '请输入菜单名称', trigger: 'blur' },
  path: { required: true, message: '请输入路径', trigger: 'blur' },
  component: { required: true, message: '请输入组件路径', trigger: 'blur' }
};

// Tree options for select
const menuSelectOptions = computed(() => {
  const transform = (items: MenuItem[]): any[] => {
    return items.map(item => ({
      label: item.name,
      key: item.id,
      children: item.children ? transform(item.children) : undefined
    }));
  };
  return transform(tableData.value);
});

async function fetchRoles() {
  const { data } = await request<any>({ url: '/api/role/list' });
  if (data?.list) {
    roleOptions.value = data.list.map((r: any) => ({ label: r.displayName, value: r.name }));
  }
}

async function fetchData() {
  loading.value = true;
  try {
    const { data } = await request<any>({ url: '/api/menu/list' });
    if (data) {
      tableData.value = data.list || [];
    }
  } finally {
    loading.value = false;
  }
}

function handleAddRoot() {
  modalType.value = 'add';
  formModel.id = 0;
  formModel.parentId = null;
  formModel.name = '';
  formModel.path = '';
  formModel.component = 'layout.base';
  formModel.order = 0;
  formModel.meta = { title: '', i18nKey: '' as App.I18n.I18nKey, icon: '', localIcon: '', hideInMenu: false };
  formModel.roles = [];
  showModal.value = true;
}

function handleAddChild(row: MenuItem) {
  handleAddRoot();
  formModel.parentId = row.id;
  formModel.path = row.path + '/';
  formModel.component = 'view.';
}

function handleEdit(row: MenuItem) {
  modalType.value = 'edit';
  formModel.id = row.id;
  // Parent ID lookup is tricky from tree data, but let's assume it's there or just let them change it
  // In a real app we might need to know ParentID explicitly from API
  formModel.name = row.name;
  formModel.path = row.path;
  formModel.component = row.component;
  formModel.order = row.order || (row.meta.order ?? 0); // Try to get order from meta if top level is missing
  formModel.meta = {
    title: row.meta.title || '',
    i18nKey: (row.meta.i18nKey || '') as App.I18n.I18nKey,
    icon: row.meta.icon || '',
    localIcon: row.meta.localIcon || '',
    hideInMenu: row.meta.hideInMenu || false
  };
  formModel.roles = [...(row.requiredRoles || row.meta?.roles || row.roles || [])];
  showModal.value = true;
}

async function handleDelete(id: number) {
  const { error } = await request({ url: `/api/menu/${id}`, method: 'delete' });
  if (!error) {
    message.success('删除成功');
    fetchData();
  }
}

function closeModal() {
  showModal.value = false;
}

async function handleSubmit() {
  formRef.value?.validate(async (errors: any) => {
    if (!errors) {
      submitLoading.value = true;
      try {
        const payload = { 
          name: formModel.name,
          path: formModel.path,
          component: formModel.component,
          order: formModel.order,
          meta: {
            title: formModel.meta.title || '',
            i18nKey: formModel.meta.i18nKey || '', // Send empty string instead of undefined to satisfy backend strictness
            icon: formModel.meta.icon || '',
            localIcon: formModel.meta.localIcon || '',
            order: formModel.order, // Sync order
            hideInMenu: formModel.meta.hideInMenu,
            roles: formModel.roles // Sync roles to meta for compatibility if needed
          },
          requiredRoles: formModel.roles,
          parentId: formModel.parentId || 0 
        };
        const url = modalType.value === 'add' ? '/api/menu' : `/api/menu/${formModel.id}`;
        const method = modalType.value === 'add' ? 'post' : 'put';
        
        const { error } = await request({ url, method, data: payload });
        if (!error) {
          message.success('操作成功');
          closeModal();
          fetchData();
        }
      } finally {
        submitLoading.value = false;
      }
    }
  });
}

onMounted(() => {
  fetchRoles();
  fetchData();
});
</script>

<style scoped></style>
