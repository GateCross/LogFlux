import type { Router } from 'vue-router';

import { generateAccessible } from '@vben/access';
import { LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { startProgress, stopProgress } from '@vben/utils';

import { accessRoutes, coreRouteNames } from '#/router/routes';
import { useAuthStore } from '#/store';

/** 首次初始化页面：系统还没有任何账号时唯一可访问的业务入口 */
const SETUP_PATH = '/auth/setup';

function isUnauthorizedError(error: unknown) {
  const err = error as {
    code?: number | string;
    message?: string;
    msg?: string;
    response?: {
      data?: {
        code?: number | string;
        message?: string;
        msg?: string;
      };
      status?: number;
    };
    status?: number;
  };
  const responseData = err?.response?.data ?? {};
  const code = err?.code ?? responseData.code;
  const message = err?.message ?? err?.msg ?? responseData.message ?? responseData.msg ?? '';

  return (
    Number(code) === 401 ||
    err?.status === 401 ||
    err?.response?.status === 401 ||
    message.includes('登录状态无效')
  );
}

function toLoginRoute(fullPath: string) {
  return {
    path: LOGIN_PATH,
    query:
      fullPath === preferences.app.defaultHomePath
        ? {}
        : { redirect: encodeURIComponent(fullPath) },
    replace: true,
  };
}

/**
 * 通用守卫：页面进度条与首次加载标记
 */
function setupCommonGuard(router: Router) {
  // 记录已经加载的页面
  const loadedPaths = new Set<string>();

  router.beforeEach((to) => {
    to.meta.loaded = loadedPaths.has(to.path);

    // 页面加载进度条
    if (!to.meta.loaded && preferences.transition.progress) {
      startProgress();
    }
    return true;
  });

  router.afterEach((to) => {
    // 记录页面是否已经加载过，避免重复播放切换动画
    loadedPaths.add(to.path);

    // 关闭页面加载进度条
    if (preferences.transition.progress) {
      stopProgress();
    }
  });
}

/**
 * 登录守卫：校验 token，首次进入时把静态路由注入 router 并生成菜单。
 * 删除 RBAC 后不再请求后端菜单，也不再按角色过滤。
 */
function setupAuthGuard(router: Router) {
  // 初始化状态在每个会话里只查询一次
  let setupChecked = false;
  let needsSetup = false;

  router.beforeEach(async (to) => {
    const accessStore = useAccessStore();
    const userStore = useUserStore();
    const authStore = useAuthStore();

    // 基本路由（登录页、404 等）不参与拦截
    if (coreRouteNames.includes(to.name as string)) {
      // 未初始化时只允许停留在初始化页，避免用户面对一个无法登录的登录页
      if (!setupChecked) {
        setupChecked = true;
        needsSetup = await authStore.checkSetupStatus();
      }
      // 已登录说明初始化早已完成，此时不能再被强制拉回初始化页
      if (needsSetup && !accessStore.accessToken) {
        return to.path === SETUP_PATH ? true : { path: SETUP_PATH, replace: true };
      }
      if (!needsSetup && to.path === SETUP_PATH) {
        // 已初始化就没必要再进初始化页
        return { path: LOGIN_PATH, replace: true };
      }

      if (to.path === LOGIN_PATH && accessStore.accessToken) {
        return decodeURIComponent(
          (to.query?.redirect as string) ||
            userStore.userInfo?.homePath ||
            preferences.app.defaultHomePath,
        );
      }
      return true;
    }

    if (!accessStore.accessToken) {
      if (to.meta.ignoreAccess) {
        return true;
      }
      return to.fullPath === LOGIN_PATH ? true : toLoginRoute(to.fullPath);
    }

    // 路由只注册一次
    if (accessStore.isAccessChecked) {
      return true;
    }

    try {
      // 已登录但用户信息未加载（刷新页面）时补拉一次
      if (!userStore.userInfo) {
        await authStore.fetchUserInfo();
      }

      // 路由为前端静态定义，模式固定为 frontend。
      // 不能读 preferences.app.accessMode：该值可从 localStorage 恢复成历史值
      // （例如删除 RBAC 前的 'backend'），会导致走后端路由分支拿到空路由表。
      const { accessibleMenus, accessibleRoutes } = await generateAccessible(
        'frontend',
        {
          roles: [],
          router,
          routes: accessRoutes,
        },
      );

      accessStore.setAccessMenus(accessibleMenus);
      accessStore.setAccessRoutes(accessibleRoutes);
      accessStore.setIsAccessChecked(true);
    } catch (err) {
      if (isUnauthorizedError(err)) {
        await authStore.logout();
        return toLoginRoute(to.fullPath);
      }

      console.error('[guard] 生成路由失败:', err);
      accessStore.setIsAccessChecked(true);
      return to.fullPath;
    }

    return { ...router.resolve(to.fullPath), replace: true };
  });
}

/**
 * 项目守卫配置
 */
function createRouterGuard(router: Router) {
  /** 通用 */
  setupCommonGuard(router);
  /** 登录校验 */
  setupAuthGuard(router);
}

export { createRouterGuard };
