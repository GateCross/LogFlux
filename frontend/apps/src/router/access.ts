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

import { findRouteTitle } from './route-title';

const forbiddenComponent = () => import('#/views/_core/fallback/forbidden.vue');

/**
 * 将 "view.xxx_yyy" 转为 "/xxx/yyy/index"
 */
function viewToPath(view: string): string {
  const raw = view.replace(/^view\./, '');
  return `/${raw.replace(/_/g, '/')}/index`;
}

function routeNameOf(name: string) {
  return name
    .split(/[-_]/)
    .filter(Boolean)
    .map((item) => item.charAt(0).toUpperCase() + item.slice(1))
    .join('');
}

function routeTitleOf(route: any, meta: Record<string, any>) {
  const key = route.name || meta.title || route.title;
  return findRouteTitle(key, meta.title, route.title) || meta.title || route.title || route.name;
}

/**
 * 递归转换 LogFlux 后端路由为 vue-vben-admin 格式
 *
 * 后端 component 格式：
 *   "layout.base$view.dashboard" → 父级 BasicLayout + 子级 /dashboard/index
 *   "layout.base"                → 纯父级 BasicLayout
 *   "view.caddy_config"          → /caddy/config/index
 *   ""                           → BasicLayout
 */
function transformLogfluxRoutes(
  routes: any[],
  pageMap: ComponentRecordType,
): RouteRecordStringComponent[] {
  if (!Array.isArray(routes)) return [];

  return routes.map((route) => {
    const meta = route.meta || {};
    const comp: string = route.component || '';

    let component: string;
    const children = route.children ? [...route.children] : [];

    if (!comp || comp.trim() === '') {
      // 无 component 且有子路由 → 不设 component，vue-vben-admin 自动加 BasicLayout
      component = children.length > 0 ? '' : 'BasicLayout';
    } else if (comp.includes('$')) {
      // "layout.base$view.xxx" → 检查视图文件是否存在
      const viewPart = comp.split('$')[1] || '';
      if (viewPart.startsWith('view.')) {
        const viewPath = viewToPath(viewPart);
        const viewFile = `${viewPath}.vue`;
        if (pageMap[viewFile]) {
          // 视图存在 → 叶子路由
          component = viewPath;
        } else {
          // 视图不存在 → 当父路由
          component = '';
        }
      } else {
        component = '';
      }
    } else if (comp.startsWith('layout.')) {
      // 纯布局路由（有子路由）→ 不设 component
      component = children.length > 0 ? '' : 'BasicLayout';
    } else if (comp.startsWith('view.')) {
      const viewPath = viewToPath(comp);
      component = pageMap[`${viewPath}.vue`] ? viewPath : '';
    } else {
      component = comp;
    }

    const transformed: any = {
      name: routeNameOf(route.name),
      path: route.path,
      meta: {
        icon: meta.icon,
        ...meta,
        hideInMenu: Boolean(meta.hideInMenu),
        order: meta.order ?? route.order ?? 0,
        title: routeTitleOf(route, meta),
      },
    };

    // 只有非空 component 才设置
    if (component) {
      transformed.component = component;
    }

    if (children.length > 0) {
      transformed.children = transformLogfluxRoutes(children, pageMap)
        .filter((r) => r.component || (r.children && r.children.length > 0));
    }
    // 避免空 children
    if (transformed.children?.length === 0) {
      delete transformed.children;
    }

    return transformed as RouteRecordStringComponent;
  });
}

async function generateAccess(options: GenerateMenuAndRoutesOptions) {
  const rawPageMap: ComponentRecordType = import.meta.glob('../views/**/*.vue');
  // 规范化 key: "../views/dashboard/index.vue" → "/dashboard/index.vue"
  const pageMap: ComponentRecordType = {};
  for (const [key, value] of Object.entries(rawPageMap)) {
    // 去掉 ../views 前缀，确保以 / 开头
    const normalized = key.replace(/^\.\.\/views\/?/, '/').replace(/^([^/])/, '/$1');
    pageMap[normalized] = value;
  }
  const layoutMap: ComponentRecordType = { BasicLayout, IFrameView };

  return await generateAccessible(preferences.app.accessMode, {
    ...options,
    fetchMenuListAsync: async () => {
      message.loading({ content: `${$t('common.loadingMenu')}...`, duration: 1.5 });
      const resp = await getUserRoutesApi();
      const routes = resp?.routes ?? (Array.isArray(resp) ? resp : []);
      if (!Array.isArray(routes) || routes.length === 0) {
        throw new Error('后端未返回可用菜单');
      }
      let transformed = transformLogfluxRoutes(routes, pageMap);
      // 过滤无效路由：无 component 且无 children 的路由无法被 vue-router 处理
      transformed = transformed.filter((r) => r.component || (r.children && r.children.length > 0));
      if (transformed.length === 0) {
        throw new Error('后端菜单无法转换为可用路由');
      }
      return transformed;
    },
    forbiddenComponent,
    layoutMap,
    pageMap,
  });
}

export { generateAccess };
