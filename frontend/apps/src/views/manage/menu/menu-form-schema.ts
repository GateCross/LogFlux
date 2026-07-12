import type { VbenFormSchema } from '#/adapter/form';

import { z } from '#/adapter/form';

import type { MenuViewItem } from './menu-utils';
import { parentSelectOptions } from './menu-utils';

export function buildMenuFormSchema(options: {
  mode: 'create' | 'edit';
  flatList: MenuViewItem[];
  editingId: number | null;
  roleOptions: Array<{ label: string; value: string }>;
}): VbenFormSchema[] {
  const { mode, flatList, editingId, roleOptions } = options;
  const parentOpts = parentSelectOptions(
    flatList,
    mode === 'edit' ? editingId : null,
  );

  return [
    {
      component: 'Select',
      fieldName: 'parentId',
      label: '上级菜单',
      componentProps: {
        allowClear: true,
        options: parentOpts,
        placeholder: '留空为一级菜单',
        class: 'w-full',
      },
    },
    {
      component: 'Input',
      fieldName: 'name',
      label: '菜单唯一标识',
      componentProps: {
        placeholder: '例如 dashboard',
      },
      rules: z.string().min(1, { message: '请输入菜单唯一标识' }),
    },
    {
      component: 'Input',
      fieldName: 'path',
      label: '路径',
      componentProps: {
        placeholder: '例如 /dashboard',
      },
      rules: z.string().min(1, { message: '请输入菜单路径' }),
    },
    {
      component: 'Input',
      fieldName: 'component',
      label: '组件',
      componentProps: {
        placeholder: '例如 layout.base 或 view.dashboard',
      },
      rules: z.string().min(1, { message: '请输入组件路径' }),
    },
    {
      component: 'Input',
      fieldName: 'title',
      label: '显示名称',
      componentProps: {
        placeholder: '默认使用菜单唯一标识',
      },
    },
    {
      component: 'Input',
      fieldName: 'i18nKey',
      label: '国际化 Key',
      componentProps: {
        placeholder: '例如 route.dashboard',
      },
    },
    {
      component: 'Input',
      fieldName: 'icon',
      label: '图标',
      componentProps: {
        placeholder: '例如 mdi:home',
      },
    },
    {
      component: 'Input',
      fieldName: 'localIcon',
      label: '本地图标',
    },
    {
      component: 'InputNumber',
      fieldName: 'order',
      label: '排序',
      componentProps: {
        class: 'w-full',
      },
    },
    {
      component: 'Select',
      fieldName: 'roles',
      label: '所需角色',
      componentProps: {
        mode: 'multiple',
        options: roleOptions,
        placeholder: '留空表示公开',
        class: 'w-full',
      },
    },
    {
      component: 'Switch',
      fieldName: 'hideInMenu',
      label: '隐藏菜单',
    },
  ];
}
