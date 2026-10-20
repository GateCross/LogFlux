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

  /** 初始化状态：initialized 为 false 表示还没有任何账号 */
  export interface SetupStatusResult {
    initialized: boolean;
  }
}

/**
 * 登录 — POST /login
 */
export async function loginApi(data: AuthApi.LoginParams) {
  return requestClient.post<AuthApi.LoginResult>('/login', data);
}

/**
 * 查询系统是否已完成初始化 — GET /setup/status
 */
export async function getSetupStatusApi() {
  return requestClient.get<AuthApi.SetupStatusResult>('/setup/status');
}

/**
 * 创建首个管理员账号 — POST /setup
 * 仅在系统还没有任何账号时可用，成功后直接返回登录令牌。
 */
export async function setupApi(data: AuthApi.LoginParams) {
  return requestClient.post<AuthApi.LoginResult>('/setup', data);
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
