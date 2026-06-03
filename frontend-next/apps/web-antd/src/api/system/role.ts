import { requestClient } from '#/api/request';

export namespace RoleApi {
  export interface RoleItem {
    [key: string]: any;
  }
}

/**
 * 获取角色列表 — GET /role/list
 */
export async function getRoleListApi() {
  return requestClient.get<RoleApi.RoleItem[]>('/role/list');
}

/**
 * 更新角色权限 — PUT /role/:id/permissions
 */
export async function updateRolePermissionsApi(id: number, permissions: string[]) {
  return requestClient.put<void>(`/role/${id}/permissions`, { permissions });
}
