import type { Recordable } from '@vben/types';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { notification } from 'antdv-next';
import { defineStore } from 'pinia';

import { getSetupStatusApi, getUserInfoApi, loginApi, setupApi } from '#/api';
import { $t } from '#/locales';
import { clearRefreshToken, setRefreshToken } from '#/store/refresh-token';
import { getDefaultAvatarDataUrl } from '#/utils/avatar';

export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();
  const router = useRouter();

  const loginLoading = ref(false);

  function parseUserPreferences(value: unknown) {
    if (!value || typeof value !== 'string') return {};
    try {
      const parsed = JSON.parse(value);
      return parsed && typeof parsed === 'object' ? parsed : {};
    } catch {
      return {};
    }
  }

  function normalizeUserInfo(userInfo: any) {
    const userPreferences = parseUserPreferences(userInfo?.preferences);
    const displayName =
      (userPreferences as any).displayName || userInfo?.username || 'LogFlux 用户';
    const username = userInfo?.username || 'LF';
    // 头像优先级：API 服务器头像 > 用户中心保存的头像 > 默认生成头像
    const avatar =
      userInfo?.avatar || (userPreferences as any).avatar || getDefaultAvatarDataUrl(username);
    return {
      ...(userPreferences as Record<string, any>),
      userId: String(userInfo?.userId ?? ''),
      username: userInfo?.username,
      realName: displayName,
      avatar,
      homePath: preferences.app.defaultHomePath,
      preferences: userInfo?.preferences || '{}',
    };
  }

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
      const loginResult = await loginApi({
        username: params.username,
        password: params.password,
      });
      const { token, refreshToken } = loginResult;

      if (token) {
        // 存储 token 到 accessStore（自动持久化到 localStorage）
        accessStore.setAccessToken(token);

        // 登录成功仅经 Refresh_Token_Store 写入 refreshToken
        if (refreshToken) {
          setRefreshToken(refreshToken);
        }

        userInfo = await fetchUserInfo();

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
          // antdv-next: ArgsProps.title（v4 曾用 message）
          title: $t('authentication.loginSuccess'),
        });
      }
    } finally {
      loginLoading.value = false;
    }

    return { userInfo };
  }

  /** 系统是否还没有任何账号，用于把用户引导到初始化页 */
  async function checkSetupStatus(): Promise<boolean> {
    try {
      const { initialized } = await getSetupStatusApi();
      return !initialized;
    } catch {
      // 查询失败时按「已初始化」处理，避免把正常用户挡在登录页外
      return false;
    }
  }

  /** 创建首个管理员账号，成功后直接进入系统 */
  async function setupAdmin(params: {
    password: string;
    username: string;
  }) {
    try {
      loginLoading.value = true;
      const { token, refreshToken } = await setupApi({
        password: params.password,
        username: params.username,
      });

      accessStore.setAccessToken(token);
      if (refreshToken) {
        setRefreshToken(refreshToken);
      }

      const userInfo = await fetchUserInfo();
      notification.success({
        description: `管理员账号已创建: ${userInfo.username}`,
        duration: 3,
        title: '初始化完成',
      });
      await router.push(preferences.app.defaultHomePath);
    } finally {
      loginLoading.value = false;
    }
  }

  async function logout(redirect: boolean = true) {
    // LogFlux 无独立登出接口，直接清理本地状态（resetAllStores 会清空 token、菜单与路由标记）
    clearRefreshToken();
    resetAllStores();
    accessStore.setLoginExpired(false);

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
    const vbenUserInfo = normalizeUserInfo(userInfo);
    userStore.setUserInfo(vbenUserInfo as any);
    return vbenUserInfo;
  }

  function $reset() {
    loginLoading.value = false;
  }

  return {
    $reset,
    authLogin,
    checkSetupStatus,
    fetchUserInfo,
    loginLoading,
    logout,
    setupAdmin,
  };
});
