import type { RequestClientConfig } from '@vben/request';

import { requestClient } from '#/api/request';

export namespace UserApi {
  /** LogFlux 用户信息 */
  export interface UserInfo {
    avatar?: string;
    buttons: string[];
    preferences: string;
    roles: string[];
    userId: string;
    username: string;
  }
}

/**
 * 获取用户信息 — GET /user/info
 */
export async function getUserInfoApi(config?: RequestClientConfig) {
  return requestClient.get<UserApi.UserInfo>('/user/info', config);
}

/**
 * 更新用户偏好 — PUT /user/preferences
 */
export async function updateUserPreferencesApi(preferences: string) {
  return requestClient.put<void>('/user/preferences', { preferences });
}
