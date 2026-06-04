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

const securityChildRouteTemplates = [
  {
    component: 'view.security_source',
    meta: { hideInMenu: true, icon: 'mdi:source-branch', title: '来源管理' },
    name: 'security_source',
    path: '/security/source',
  },
  {
    component: 'view.security_policy',
    meta: { icon: 'mdi:shield-check', title: '策略管理' },
    name: 'security_policy',
    path: '/security/policy',
  },
  {
    component: 'view.security_observe',
    meta: { icon: 'mdi:eye-outline', title: '观测日志' },
    name: 'security_observe',
    path: '/security/observe',
  },
  {
    component: 'view.security_ops',
    meta: { hideInMenu: true, icon: 'mdi:wrench-outline', title: '运维操作' },
    name: 'security_ops',
    path: '/security/ops',
  },
  {
    component: 'view.security_runtime',
    meta: { hideInMenu: true, icon: 'mdi:clock-outline', title: '运行时' },
    name: 'security_runtime',
    path: '/security/runtime',
  },
  {
    component: 'view.security_crs',
    meta: { hideInMenu: true, icon: 'mdi:tune-vertical', title: 'CRS 调优' },
    name: 'security_crs',
    path: '/security/crs',
  },
  {
    component: 'view.security_exclusion',
    meta: { hideInMenu: true, icon: 'mdi:cancel', title: '排除规则' },
    name: 'security_exclusion',
    path: '/security/exclusion',
  },
  {
    component: 'view.security_binding',
    meta: { hideInMenu: true, icon: 'mdi:link-variant', title: '绑定管理' },
    name: 'security_binding',
    path: '/security/binding',
  },
  {
    component: 'view.security_release',
    meta: { hideInMenu: true, icon: 'mdi:rocket-launch-outline', title: '发布管理' },
    name: 'security_release',
    path: '/security/release',
  },
  {
    component: 'view.security_job',
    meta: { hideInMenu: true, icon: 'mdi:timer-outline', title: '任务管理' },
    name: 'security_job',
    path: '/security/job',
  },
];

const routeTitleMap: Record<string, string> = {
  caddy: 'Caddy 配置',
  caddy_access: '访问控制',
  caddy_config: '配置管理',
  caddy_log: '访问日志',
  caddy_source: '来源管理',
  'caddy_system-log': '系统日志',
  caddy_system_log: '系统日志',
  cron: '定时任务',
  dashboard: '仪表盘',
  manage: '系统管理',
  manage_menu: '菜单管理',
  manage_role: '角色管理',
  manage_user: '用户管理',
  notification: '通知管理',
  notification_channel: '通知渠道',
  notification_log: '发送日志',
  notification_rule: '通知规则',
  notification_template: '通知模板',
  security: '安全管理',
  security_binding: '绑定管理',
  security_crs: 'CRS 调优',
  security_exclusion: '排除规则',
  security_job: '任务管理',
  security_observe: '观测日志',
  security_ops: '运维操作',
  security_policy: '策略管理',
  security_release: '发布管理',
  security_runtime: '运行时',
  security_source: '来源管理',
  user: '个人中心',
  user_center: '个人中心',
};

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
  return routeTitleMap[key] || routeTitleMap[meta.title] || meta.title || route.title || route.name;
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
    if (route.name === 'security') {
      const childNames = new Set(children.map((child: any) => child.name));
      children.push(
        ...securityChildRouteTemplates.filter((child) => !childNames.has(child.name)),
      );
    }

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
      name: routeNameOf(route.name),
      path: route.path,
      meta: {
        icon: meta.icon,
        ...meta,
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
