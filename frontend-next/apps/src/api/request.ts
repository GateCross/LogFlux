/**
 * LogFlux 请求客户端配置
 * 适配 LogFlux 后端的响应格式和认证机制
 */
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

import { message } from 'ant-design-vue';

import { useAuthStore } from '#/store';

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
    accessStore.setAccessToken(null);
    if (
      preferences.app.loginExpiredMode === 'modal' &&
      accessStore.isAccessChecked
    ) {
      accessStore.setLoginExpired(true);
    } else {
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
      const resp = await refreshTokenApi();
      const newToken = resp?.token ?? null;
      if (newToken) {
        accessStore.setAccessToken(newToken);
        // 同时更新 refreshToken（如果返回了的话）
        const newRefreshToken = resp?.refreshToken;
        if (newRefreshToken) {
          accessStore.setRefreshToken?.(newRefreshToken);
          // 同步到 LF_ 前缀 key，供 refreshTokenApi 读取
          localStorage.setItem('LF_refreshToken', newRefreshToken);
        }
        return newToken;
      }
      return '';
    } catch {
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

  // 通用的错误处理
  // LogFlux 错误响应：{ code: 4xx/5xx, message: "错误描述" }
  client.addResponseInterceptor(
    errorMessageResponseInterceptor((msg: string, error) => {
      const responseData = error?.response?.data ?? {};
      // LogFlux 使用 message 或 msg 字段（后端兼容旧前端）
      const errorMessage =
        responseData?.message ?? responseData?.msg ?? responseData?.error ?? '';
      message.error(errorMessage || msg);
    }),
  );

  return client;
}

export const requestClient = createRequestClient(apiURL, {
  responseReturn: 'data',
});

export const baseRequestClient = new RequestClient({ baseURL: apiURL });
