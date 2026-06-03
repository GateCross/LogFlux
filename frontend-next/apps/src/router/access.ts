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
 * 将 "view.xxx_yyy" 转为 "/xxx/yyy/index"
 */
function viewToPath(view: string): string {
  const raw = view.replace(/^view\./, '');
  return `/${raw.replace(/_/g, '/')}/index`;
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
      component = viewToPath(comp);
    } else {
      component = comp;
    }

    const transformed: any = {
      name: route.name,
      path: route.path,
      meta: {
        icon: meta.icon,
        title: meta.title || route.name,
        order: meta.order || 0,
        ...meta,
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
  console.log('[access] pageMap keys:', Object.keys(pageMap).slice(0, 10));
  const layoutMap: ComponentRecordType = { BasicLayout, IFrameView };

  return await generateAccessible(preferences.app.accessMode, {
    ...options,
    fetchMenuListAsync: async () => {
      message.loading({ content: `${$t('common.loadingMenu')}...`, duration: 1.5 });
      try {
        const resp = await getUserRoutesApi();
        const routes = resp?.routes ?? (Array.isArray(resp) ? resp : []);
        if (!Array.isArray(routes) || routes.length === 0) {
          console.warn('[access] 路由为空:', resp);
          return [];
        }
        let transformed = transformLogfluxRoutes(routes, pageMap);
        // 过滤无效路由：无 component 且无 children 的路由无法被 vue-router 处理
        transformed = transformed.filter((r) => r.component || (r.children && r.children.length > 0));
        console.log('[access] 有效路由数:', transformed.length);
        return transformed;
      } catch (err: any) {
        console.error('[access] 获取路由失败:', err?.message || err);
        message.error(`获取菜单失败: ${err?.message || '未知错误'}`);
        return [];
      }
    },
    forbiddenComponent,
    layoutMap,
    pageMap,
  });
}

export { generateAccess };
