/**
 * 鉴权 Service_API（迁移自旧 Vue 版 `frontend/src/service/api/auth.ts`）。
 *
 * 迁移说明（task 4.1 / Req 3.1、3.2、3.7）：
 *  - 端点与旧版逐一对齐：`POST /api/login`、`GET /api/user/info`、
 *    `PUT /api/user/preferences`、`POST /api/refreshToken`，与 `backend/api/auth.api`
 *    及 `backend/api/user.api` 契约一致。
 *  - 全部调用统一经 Request_Layer（`src/utils/request`），返回扁平结构 `{ data, error }`；
 *    Request_Layer 已配置 10s 默认超时（Req 3.1、3.2、3.7）、鉴权头注入与响应码处理，
 *    本层只负责声明端点与请求体，不重复处理副作用。
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

/**
 * 登录（`POST /api/login`，Req 3.1）。
 *
 * @param username 用户名
 * @param password 密码（调用方应先经 `@/utils/crypto#encrypt` 加密）
 * @returns 登录令牌对 `{ token, refreshToken }`
 */
export function fetchLogin(username: string, password: string): Promise<FlatResponse<Api.Auth.LoginToken>> {
  return request<Api.Auth.LoginToken>({
    url: '/api/login',
    method: 'post',
    data: {
      username,
      password,
    },
  });
}

/**
 * 获取当前登录用户信息与角色集合（`GET /api/user/info`，Req 3.2）。
 *
 * @returns 用户信息（含 `roles`、可选 `preferences`）
 */
export function fetchGetUserInfo(): Promise<FlatResponse<Api.Auth.UserInfo>> {
  return request<Api.Auth.UserInfo>({ url: '/api/user/info' });
}

/**
 * 更新用户偏好（`PUT /api/user/preferences`）。
 *
 * @param preferences 用户偏好 JSON 字符串（由 Preferences_Store 序列化）
 */
export function fetchUpdateUserPreferences(preferences: string): Promise<FlatResponse<void>> {
  return request<void>({
    url: '/api/user/preferences',
    method: 'put',
    data: {
      preferences,
    },
  });
}

/**
 * 刷新令牌（`POST /api/refreshToken`，Req 3.7）。
 *
 * @param refreshToken 刷新令牌
 * @returns 新的登录令牌对 `{ token, refreshToken }`
 */
export function fetchRefreshToken(refreshToken: string): Promise<FlatResponse<Api.Auth.LoginToken>> {
  return request<Api.Auth.LoginToken>({
    url: '/api/refreshToken',
    method: 'post',
    data: {
      refreshToken,
    },
  });
}
