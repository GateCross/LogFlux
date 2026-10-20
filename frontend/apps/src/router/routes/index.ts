import type { RouteRecordRaw } from 'vue-router';

import { mergeRouteModules, traverseTreeValues } from '@vben/utils';

import { coreRoutes, fallbackNotFoundRoute } from './core';

const routeFiles = import.meta.glob('./modules/**/*.ts', { eager: true });

/** 业务路由：静态定义，删除 RBAC 后不再由后端按角色下发 */
const accessRoutes: RouteRecordRaw[] = mergeRouteModules(routeFiles);

/** 顶层路由：基本路由 + 404 兜底，业务路由由 generateAccessible 注入 BasicLayout */
const routes: RouteRecordRaw[] = [
  ...coreRoutes,
  fallbackNotFoundRoute,
];

/** 基本路由名称，这些路由不参与登录拦截 */
const coreRouteNames = traverseTreeValues(coreRoutes, (route) => route.name);

export { accessRoutes, coreRouteNames, routes };
