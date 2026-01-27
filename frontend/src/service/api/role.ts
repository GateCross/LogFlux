import { request } from '../request';

/**
 * Get role list
 */
export function fetchGetRoleList() {
    return request<Api.Role.RoleListResp>({ url: '/api/role/list' });
}

/**
 * Update role permissions
 * @param id Role ID
 * @param permissions Permission list
 */
export function fetchUpdateRolePermissions(id: number, permissions: string[]) {
    return request<any>({
        url: `/api/role/${id}/permissions`,
        method: 'put',
        data: { id, permissions }
    });
}
