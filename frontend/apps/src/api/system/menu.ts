import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace MenuApi {
  export interface RouteMeta {
    hideInMenu?: boolean;
    i18nKey?: string;
    icon?: string;
    localIcon?: string;
    order?: number;
    roles?: string[];
    title?: string;
  }

  export interface MenuItem {
    children?: MenuItem[];
    component: string;
    createdAt?: string;
    id: number;
    meta: RouteMeta;
    name: string;
    order: number;
    parentId: number;
    path: string;
    requiredRoles: string[];
  }

  export interface MenuListResult {
    list: MenuItem[];
  }

  export interface MenuPayload {
    component: string;
    meta: RouteMeta;
    name: string;
    order: number;
    parentId: number;
    path: string;
    requiredRoles: string[];
  }
}

export async function getMenuListApi(config?: RequestClientConfig) {
  const resp = await requestClient.get<MenuApi.MenuListResult>(
    '/menu/list',
    config,
  );
  return listOf(resp);
}

export async function createMenuApi(data: MenuApi.MenuPayload) {
  return requestClient.post<void>('/menu', data);
}

export async function updateMenuApi(id: number, data: MenuApi.MenuPayload) {
  return requestClient.put<void>(`/menu/${id}`, data);
}

export async function deleteMenuApi(id: number) {
  return requestClient.delete<void>(`/menu/${id}`);
}
