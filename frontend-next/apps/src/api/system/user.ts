import { requestClient } from '#/api/request';

export namespace UserManageApi {
  export interface UserItem {
    id: number;
    username: string;
    roles: string[];
    status: number;
    createdAt: string;
  }

  export interface UserListResp {
    list: UserItem[];
    total: number;
  }

  export interface CreateUserParams {
    username: string;
    password: string;
    roles: string[];
  }

  export interface UpdateUserParams {
    password?: string;
    roles?: string[];
  }
}

/** 获取用户列表 — GET /user/list */
export async function getUserListApi(params?: {
  page?: number;
  pageSize?: number;
  username?: string;
}) {
  return requestClient.get<UserManageApi.UserListResp>('/user/list', {
    params,
  });
}

/** 创建用户 — POST /user */
export async function createUserApi(data: UserManageApi.CreateUserParams) {
  return requestClient.post<void>('/user', data);
}

/** 更新用户 — PUT /user/:id */
export async function updateUserApi(
  id: number,
  data: UserManageApi.UpdateUserParams,
) {
  return requestClient.put<void>(`/user/${id}`, data);
}

/** 删除用户 — DELETE /user/:id */
export async function deleteUserApi(id: number) {
  return requestClient.delete<void>(`/user/${id}`);
}

/** 切换用户状态 — PUT /user/:id/status */
export async function toggleUserStatusApi(id: number) {
  return requestClient.put<void>(`/user/${id}/status`);
}
