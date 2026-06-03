import type {
  ComponentRecordType,
  GenerateMenuAndRoutesOptions,
  RouteRecordStringComponent,
} from '@vben/types';

import { generateAccessible } from '@vben/access';
import { preferences } from '@vben/preferences';

import { message } from 'ant-design-vue';

import { getUserRoutesApi } from '#/api';
import { BasicLayout, IFrameView } from '#/layouts';
import { $t } from '#/locales';

const forbiddenComponent = () => import('#/views/_core/fallback/forbidden.vue');

/**
 * 将 LogFlux 后端的 component 字段转换为 vue-vben-admin 期望的路径格式
 *
 * LogFlux 后端返回: "view.caddy_config" → 转换为 "/caddy/config/index"
 * LogFlux 后端返回: "view.security" → 转换为 "/security/index"（父级路由）
 * 后端返回空/无 component 的父级路由 → 使用 "BasicLayout"
 */
function transformLogfluxComponent(component: string | undefined): string {
  if (!component) return 'BasicLayout';

  // "view.caddy_config" → "caddy_config" → "caddy/config" → "/caddy/config/index"
  const raw = component.replace(/^view\./, '');
  const path = raw.replace(/_/g, '/');
  return `/${path}/index`;
}

/**
 * 递归转换 LogFlux 路由为 vue-vben-admin RouteRecordStringComponent 格式
 */
function transformLogfluxRoutes(
  routes: any[],
  isRoot = true,
): RouteRecordStringComponent[] {
  return routes.map((route) => {
    const transformed: any = {
      name: route.name,
      path: route.path,
      component: isRoot
        ? 'BasicLayout'
        : transformLogfluxComponent(route.component),
      meta: {
        icon: route.icon || route.meta?.icon,
        title: route.title || route.meta?.title || route.name,
        order: route.order ?? route.meta?.order,
        ...(route.meta || {}),
      },
    };

    if (route.children?.length) {
      transformed.children = transformLogfluxRoutes(route.children, false);
    }

    return transformed as RouteRecordStringComponent;
  });
}

/**
 * 安全子路由模板 — 确保 security 父路由始终包含所有子页面
 * 即使后端没有返回某些子路由，前端也会补全
 */
const SECURITY_CHILDREN_TEMPLATES = [
  { name: 'security_source', path: '/security/source', title: 'security_source' },
  { name: 'security_policy', path: '/security/policy', title: 'security_policy' },
  { name: 'security_observe', path: '/security/observe', title: 'security_observe' },
  { name: 'security_ops', path: '/security/ops', title: 'security_ops' },
  { name: 'security_runtime', path: '/security/runtime', title: 'security_runtime' },
  { name: 'security_crs', path: '/security/crs', title: 'security_crs' },
  { name: 'security_exclusion', path: '/security/exclusion', title: 'security_exclusion' },
  { name: 'security_binding', path: '/security/binding', title: 'security_binding' },
  { name: 'security_release', path: '/security/release', title: 'security_release' },
  { name: 'security_job', path: '/security/job', title: 'security_job' },
];

/**
 * 确保 security 父路由包含所有必需的子路由
 */
function normalizeSecurityRoutes(
  routes: RouteRecordStringComponent[],
): RouteRecordStringComponent[] {
  return routes.map((route) => {
    const normalized = { ...route };

    if (normalized.children?.length) {
      normalized.children = normalizeSecurityRoutes(normalized.children);
    }

    if (normalized.name !== 'security') {
      return normalized;
    }

    // 确保 security 路由有 BasicLayout 组件
    normalized.component = 'BasicLayout';

    const existingNames = new Set(
      (normalized.children || []).map((c: any) => c.name),
    );
    const missing = SECURITY_CHILDREN_TEMPLATES.filter(
      (t) => !existingNames.has(t.name),
    );

    if (missing.length > 0) {
      normalized.children = [
        ...(normalized.children || []),
        ...missing.map((t) => ({
          name: t.name,
          path: t.path,
          component: transformLogfluxComponent(`view.${t.name}`),
          meta: {
            title: t.title,
            hideInMenu: true,
          },
        })),
      ] as RouteRecordStringComponent[];
    }

    return normalized;
  });
}

async function generateAccess(options: GenerateMenuAndRoutesOptions) {
  const pageMap: ComponentRecordType = import.meta.glob('../views/**/*.vue');

  const layoutMap: ComponentRecordType = {
    BasicLayout,
    IFrameView,
  };

  return await generateAccessible(preferences.app.accessMode, {
    ...options,
    fetchMenuListAsync: async () => {
      message.loading({
        content: `${$t('common.loadingMenu')}...`,
        duration: 1.5,
      });

      // 从 LogFlux 后端获取动态路由
      const { routes } = await getUserRoutesApi();

      // 转换为 vue-vben-admin 格式
      let transformedRoutes = transformLogfluxRoutes(routes);

      // 补全 security 子路由
      transformedRoutes = normalizeSecurityRoutes(transformedRoutes);

      return transformedRoutes;
    },
    forbiddenComponent,
    layoutMap,
    pageMap,
  });
}

export { generateAccess };
