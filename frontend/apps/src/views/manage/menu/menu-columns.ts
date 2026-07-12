import type { VxeTableGridColumns } from '#/adapter/vxe-table';

import type { MenuViewItem } from './menu-utils';

export const menuGridColumns: VxeTableGridColumns<MenuViewItem> = [
  {
    field: 'displayTitle',
    title: '菜单名称',
    width: 260,
    treeNode: true,
    align: 'left',
    headerAlign: 'center',
    slots: { default: 'title' },
  },
  { field: 'path', title: '路径', minWidth: 180, showOverflow: true },
  { field: 'component', title: '组件', minWidth: 220, showOverflow: true },
  { field: 'order', title: '排序', width: 80 },
  {
    field: 'i18nKey',
    title: '国际化 Key',
    minWidth: 180,
    showOverflow: true,
  },
  {
    field: 'roles',
    title: '所需角色',
    width: 220,
    slots: { default: 'roles' },
  },
  {
    field: 'hideInMenu',
    title: '菜单显示',
    width: 100,
    slots: { default: 'hideInMenu' },
  },
  {
    field: 'action',
    title: '操作',
    width: 280,
    fixed: 'right',
    slots: { default: 'action' },
  },
];
