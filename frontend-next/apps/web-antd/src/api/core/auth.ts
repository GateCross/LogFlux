import { baseRequestClient, requestClient } from '#/api/request';

export namespace AuthApi {
  /** 登录接口参数 */
  export interface LoginParams {
    password: string;
    username: string;
  }

  /** 登录接口返回值 — LogFlux 返回 token + refreshToken */
  export interface LoginResult {
    refreshToken: string;
    token: string;
  }

  /** 刷新 token 返回值 */
  export interface RefreshTokenResult {
    refreshToken: string;
    token: string;
  }
}

/**
 * 登录 — POST /login
 */
export async function loginApi(data: AuthApi.LoginParams) {
  return requestClient.post<AuthApi.LoginResult>('/login', data);
}

/**
 * 刷新 accessToken — POST /refreshToken
 * LogFlux 使用 body 传递 refreshToken，非 cookie
 */
export async function refreshTokenApi() {
  const refreshToken = localStorage.getItem('LF_refreshToken') || '';
  return baseRequestClient.post<AuthApi.RefreshTokenResult>('/refreshToken', {
    refreshToken,
  });
}

/**
 * 获取用户权限码（LogFlux 无独立接口，从 user/info 中获取 roles）
 */
export async function getAccessCodesApi() {
  // LogFlux 没有独立的 /auth/codes 接口
  // 权限码从用户信息中的 roles 派生，此处返回空数组
  // 实际权限控制通过路由守卫中的 roles 判断
  return [] as string[];
}
