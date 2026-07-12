import type { MenuApi } from '#/api/system/menu';

import { findRouteTitle } from '#/router/route-title';

export interface MenuViewItem extends MenuApi.MenuItem {
  children?: MenuViewItem[];
  displayTitle: string;
  hideInMenu: boolean;
  i18nKey: string;
  icon: string;
  localIcon: string;
  roles: string[];
  title: string;
  type: 'directory' | 'menu';
}

export interface MenuFormValues extends Record<string, unknown> {
  component: string;
  hideInMenu: boolean;
  i18nKey: string;
  icon: string;
  localIcon: string;
  name: string;
  order: number;
  /** 0 / null / undefined 表示一级菜单 */
  parentId: number | null;
  path: string;
  roles: string[];
  title: string;
}

export function resolveMenuTitle(
  item: MenuApi.MenuItem,
  rawTitle: string,
): string {
  return (
    findRouteTitle(item.name, rawTitle, item.meta?.i18nKey) ||
    rawTitle ||
    item.name
  );
}

export function normalizeMenuItem(item: MenuApi.MenuItem): MenuViewItem {
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
    roles: item.requiredRoles?.length
      ? item.requiredRoles
      : (meta.roles ?? []),
    title: rawTitle,
    type:
      children.length > 0 || item.component.startsWith('layout.')
        ? 'directory'
        : 'menu',
  };
}

export function flattenTree(
  nodes: MenuViewItem[],
  result: MenuViewItem[] = [],
): MenuViewItem[] {
  for (const node of nodes) {
    result.push(node);
    if (node.children?.length) {
      flattenTree(node.children, result);
    }
  }
  return result;
}

export function createMenuFormDefaults(
  parentId: number | null = 0,
): MenuFormValues {
  const pid = parentId ?? 0;
  return {
    component: pid > 0 ? 'view.' : 'layout.base',
    hideInMenu: false,
    i18nKey: '',
    icon: '',
    localIcon: '',
    name: '',
    order: 0,
    parentId: pid,
    path: pid > 0 ? '/' : '',
    roles: [],
    title: '',
  };
}

export function mapMenuRecordToFormValues(record: MenuViewItem): MenuFormValues {
  return {
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
  };
}

export function buildMenuPayload(values: MenuFormValues): MenuApi.MenuPayload {
  const title =
    (typeof values.title === 'string' && values.title.trim()) ||
    String(values.name ?? '');
  const roles = Array.isArray(values.roles) ? values.roles : [];
  return {
    component: String(values.component ?? '').trim(),
    meta: {
      hideInMenu: Boolean(values.hideInMenu),
      i18nKey: String(values.i18nKey ?? ''),
      icon: String(values.icon ?? ''),
      localIcon: String(values.localIcon ?? ''),
      order: Number(values.order) || 0,
      roles,
      title,
    },
    name: String(values.name ?? '').trim(),
    order: Number(values.order) || 0,
    parentId: values.parentId ?? 0,
    path: String(values.path ?? '').trim(),
    requiredRoles: roles,
  };
}

export function childCreateOverrides(parent: MenuViewItem): Partial<MenuFormValues> {
  return {
    parentId: parent.id,
    path: `${parent.path.replace(/\/$/, '')}/`,
    component: 'view.',
  };
}

export function parentSelectOptions(
  flatList: MenuViewItem[],
  excludeId: number | null,
): Array<{ label: string; value: number }> {
  return flatList
    .filter((item) => item.id !== excludeId)
    .map((item) => ({
      label: `${item.displayTitle} (${item.path})`,
      value: item.id,
    }));
}
