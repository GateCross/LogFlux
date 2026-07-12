/** LogFlux 请求客户端：鉴权、响应格式与错误 toast */
import type { RequestClientOptions } from '@vben/request';

import { useAppConfig } from '@vben/hooks';
import { preferences } from '@vben/preferences';
import {
  authenticateResponseInterceptor,
  defaultResponseInterceptor,
  errorMessageResponseInterceptor,
  RequestClient,
} from '@vben/request';
import { useAccessStore } from '@vben/stores';

import { message } from 'antdv-next';

import { useAuthStore } from '#/store';
import { clearRefreshToken, setRefreshToken } from '#/store/refresh-token';
import { apiErrorMessage } from '#/utils/api-error-message';

import { refreshTokenApi } from './core';

const { apiURL } = useAppConfig(import.meta.env, import.meta.env.PROD);

function createRequestClient(baseURL: string, options?: RequestClientOptions) {
  const client = new RequestClient({
    ...options,
    baseURL,
  });

  /**
   * 重新认证逻辑 — token 完全失效，跳转登录
   */
  async function doReAuthenticate() {
    console.warn('Access token or refresh token is invalid or expired.');
    const accessStore = useAccessStore();
    const authStore = useAuthStore();
    const isAccessChecked = accessStore.isAccessChecked;
    clearRefreshToken();
    accessStore.setAccessToken(null);

    if (
      preferences.app.loginExpiredMode === 'modal' &&
      isAccessChecked
    ) {
      accessStore.setLoginExpired(true);
    } else if (isAccessChecked) {
      await authStore.logout();
    }
  }

  /**
   * 刷新 token 逻辑
   * LogFlux 后端：POST /refreshToken，新 token 在响应 body.data.token 中返回
   */
  async function doRefreshToken() {
    const accessStore = useAccessStore();
    try {
      const rawResp = await refreshTokenApi();
      const resp = (rawResp as any)?.data?.data ?? (rawResp as any)?.data ?? rawResp;
      const newToken = resp?.token ?? null;
      if (newToken) {
        accessStore.setAccessToken(newToken);
        // 刷新成功仅经 Refresh_Token_Store 写入 refreshToken
        const newRefreshToken = resp?.refreshToken;
        if (newRefreshToken) {
          setRefreshToken(newRefreshToken);
        }
        return newToken;
      }
      // 刷新失败：清理 access + refresh
      clearRefreshToken();
      accessStore.setAccessToken(null);
      return '';
    } catch {
      clearRefreshToken();
      accessStore.setAccessToken(null);
      return '';
    }
  }

  function formatToken(token: null | string) {
    return token ? `Bearer ${token}` : null;
  }

  // 请求头处理 — 注入 JWT token
  client.addRequestInterceptor({
    fulfilled: async (config) => {
      const accessStore = useAccessStore();
      config.headers.Authorization = formatToken(accessStore.accessToken);
      config.headers['Accept-Language'] = preferences.app.locale;
      return config;
    },
  });

  // LogFlux 部分接口会用 HTTP 200 + body.code=401 表示登录失效
  client.addResponseInterceptor({
    fulfilled: async (response) => {
      if (
        response.config.responseReturn !== 'raw' &&
        Number(response.data?.code) === 401
      ) {
        await doReAuthenticate();
        throw Object.assign({}, response, { response });
      }
      return response;
    },
  });

  // 处理返回的响应数据格式
  // LogFlux 返回 { code: 0, data: T, message: "成功" }
  client.addResponseInterceptor(
    defaultResponseInterceptor({
      codeField: 'code',
      dataField: 'data',
      successCode: 0,
    }),
  );

  // token 过期的处理
  client.addResponseInterceptor(
    authenticateResponseInterceptor({
      client,
      doReAuthenticate,
      doRefreshToken,
      enableRefreshToken: preferences.app.enableRefreshToken,
      formatToken,
    }),
  );

  // 通用的错误处理（全局 toast ≤1；errorMessageMode: 'none' 时不 toast）
  // LogFlux 错误响应：{ code: 4xx/5xx, message: "错误描述" }
  client.addResponseInterceptor(
    errorMessageResponseInterceptor((msg: string, error) => {
      // 文案统一经 apiErrorMessage（双参 fallback）；拦截器层已保证同一失败只回调一次
      message.error(apiErrorMessage(error, msg));
    }),
  );

  return client;
}

export const requestClient = createRequestClient(apiURL, {
  responseReturn: 'data',
});

export const baseRequestClient = new RequestClient({ baseURL: apiURL });

/** 抑制全局错误 toast（errorMessageMode: 'none'） */
export const suppressGlobalErrorToast = {
  errorMessageMode: 'none',
} as const;
