import { requestClient } from '#/api/request';

import { listOf } from '../_utils';

export namespace RoleApi {
  export interface RoleItem {
    createdAt?: string;
    description?: string;
    displayName?: string;
    id: number;
    name: string;
    permissions?: string[];
  }

  export interface RoleListResult {
    list: RoleItem[];
  }
}

/**
 * 获取角色列表 — GET /role/list
 */
export async function getRoleListApi() {
  const resp = await requestClient.get<RoleApi.RoleListResult>('/role/list');
  return listOf(resp);
}

/**
 * 更新角色权限 — PUT /role/:id/permissions
 */
export async function updateRolePermissionsApi(id: number, permissions: string[]) {
  return requestClient.put<void>(`/role/${id}/permissions`, { permissions });
}
