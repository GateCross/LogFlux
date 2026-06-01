/**
 * Service_API：角色模块（对齐 backend/api/role.api，Req 17.3 / 11.2）。
 *
 * 迁移自旧 Vue 版 `frontend/src/service/api/role.ts`，统一经 Request_Layer
 * 返回扁平 `{ data, error }`。`Api.Role.*` 类型见 `src/typings.d.ts`。
 */
import { request } from '@/utils/request';

/** GET /api/role/list —— 获取角色列表。 */
export function fetchGetRoleList() {
  return request<Api.Role.RoleListResp>({ url: '/api/role/list' });
}

/**
 * PUT /api/role/:id/permissions —— 更新角色权限。
 *
 * @param id 角色 ID
 * @param permissions 权限列表
 */
export function fetchUpdateRolePermissions(id: number, permissions: string[]) {
  return request<void>({
    url: `/api/role/${id}/permissions`,
    method: 'put',
    data: { id, permissions },
  });
}
