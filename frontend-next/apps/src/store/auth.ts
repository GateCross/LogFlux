import type { Recordable } from '@vben/types';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { notification } from 'ant-design-vue';
import { defineStore } from 'pinia';

import { getAccessCodesApi, getUserInfoApi, loginApi } from '#/api';
import { $t } from '#/locales';
import { encrypt } from '#/utils/crypto';

export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();
  const router = useRouter();

  const loginLoading = ref(false);

  /**
   * 异步处理登录操作
   * LogFlux: POST /login → { token, refreshToken }
   */
  async function authLogin(
    params: Recordable<any>,
    onSuccess?: () => Promise<void> | void,
  ) {
    let userInfo: any = null;
    try {
      loginLoading.value = true;

      // LogFlux 登录返回 { token, refreshToken }
      // 密码需要 AES 加密后发送
      const loginResult = await loginApi({
        username: params.username,
        password: encrypt(params.password),
      });
      const { token, refreshToken } = loginResult;

      if (token) {
        // 存储 token 到 accessStore（自动持久化到 localStorage）
        accessStore.setAccessToken(token);

        // 存储 refreshToken
        if (refreshToken) {
          accessStore.setRefreshToken?.(refreshToken);
          // 同时存储到 LF_ 前缀的 key，供 refreshTokenApi 使用
          localStorage.setItem('LF_refreshToken', refreshToken);
        }

        // 获取用户信息和权限码
        const [fetchUserInfoResult, accessCodes] = await Promise.all([
          fetchUserInfo(),
          getAccessCodesApi(),
        ]);

        userInfo = fetchUserInfoResult;

        // 映射 LogFlux 用户信息到 vben 格式
        const vbenUserInfo = {
          userId: userInfo.userId,
          username: userInfo.username,
          realName: userInfo.username,
          roles: userInfo.roles || [],
          homePath: preferences.app.defaultHomePath,
        };
        userStore.setUserInfo(vbenUserInfo as any);
        accessStore.setAccessCodes(accessCodes);

        if (accessStore.loginExpired) {
          accessStore.setLoginExpired(false);
        } else {
          onSuccess
            ? await onSuccess?.()
            : await router.push(preferences.app.defaultHomePath);
        }

        notification.success({
          description: `${$t('authentication.loginSuccessDesc')}: ${userInfo.username}`,
          duration: 3,
          message: $t('authentication.loginSuccess'),
        });
      }
    } finally {
      loginLoading.value = false;
    }

    return { userInfo };
  }

  async function logout(redirect: boolean = true) {
    // LogFlux 无独立登出接口，直接清理本地状态
    resetAllStores();
    accessStore.setLoginExpired(false);

    // 清理 LF_ 前缀的 refreshToken
    localStorage.removeItem('LF_refreshToken');

    await router.replace({
      path: LOGIN_PATH,
      query: redirect
        ? {
            redirect: encodeURIComponent(
              router.currentRoute.value.fullPath,
            ),
          }
        : {},
    });
  }

  async function fetchUserInfo() {
    const userInfo = await getUserInfoApi();

    // 映射到 vben 格式
    const vbenUserInfo = {
      userId: userInfo.userId,
      username: userInfo.username,
      realName: userInfo.username,
      roles: userInfo.roles || [],
      homePath: preferences.app.defaultHomePath,
    };
    userStore.setUserInfo(vbenUserInfo as any);
    return userInfo;
  }

  function $reset() {
    loginLoading.value = false;
  }

  return {
    $reset,
    authLogin,
    fetchUserInfo,
    loginLoading,
    logout,
  };
});
