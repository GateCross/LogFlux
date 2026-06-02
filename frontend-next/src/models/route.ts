/**
 * 路由模型（Route_Module，任务 7.1）。
 *
 * 设计依据：design.md「Route_Module」，迁移自旧 Vue 版 `frontend/src/store/modules/route/index.ts`。
 *
 * 职责：
 *  - 常量路由初始化（`initConstantRoute`）：拉取常量路由、过滤内置路由、规范化安全路由。
 *  - 动态路由初始化（`initAuthRoute`）：拉取用户动态路由、规范化安全路由、生成菜单树。
 *  - RBAC 访问控制（`canAccess`，Req 4.5）：超级角色放行，否则校验角色交集。
 *  - 路由树到菜单树转换（`routeTreeToMenuTree`，Req 4.3）：过滤隐藏项、保持层级与排序。
 *  - 安全路由规范化（`normalizeSecurityRoutes`）：补全缺失子路由与默认重定向。
 *
 * 支撑的需求：4.3, 4.5, 4.8
 */
import { useState, useCallback } from 'react';
import {
  fetchGetConstantRoutes,
  fetchGetUserRoutes,
  fetchIsRouteExist,
} from '@/services/route';
import { SUPER_ROLE, ROUTE_HOME } from '@/constants/app';

// ──────────────────────────────────────────────────────────────────────────
// 菜单项类型
// ──────────────────────────────────────────────────────────────────────────

export interface MenuItem {
  /** 路由路径，ProLayout 用作导航 key */
  path: string;
  /** 显示名称，ProLayout 用作菜单文本 */
  name: string;
  i18nKey?: string;
  icon?: string;
  children?: MenuItem[];
  href?: string;
  hideInMenu?: boolean;
}

// ──────────────────────────────────────────────────────────────────────────
// 安全模块子路由模板
// ──────────────────────────────────────────────────────────────────────────

const securityChildRouteTemplates: Api.Route.MenuRoute[] = [
  {
    name: 'security_source',
    path: '/security/source',
    meta: { title: 'security_source', i18nKey: 'route.security_source', icon: 'carbon:catalog' },
  },
  {
    name: 'security_policy',
    path: '/security/policy',
    meta: { title: 'security_policy', i18nKey: 'route.security_policy', icon: 'carbon:settings' },
  },
  {
    name: 'security_observe',
    path: '/security/observe',
    meta: { title: 'security_observe', i18nKey: 'route.security_observe', icon: 'carbon:view' },
  },
  {
    name: 'security_ops',
    path: '/security/ops',
    meta: { title: 'security_ops', i18nKey: 'route.security_ops', icon: 'carbon:time' },
  },
  {
    name: 'security_runtime',
    path: '/security/runtime',
    meta: { title: 'security_runtime', i18nKey: 'route.security_runtime', hideInMenu: true },
  },
  {
    name: 'security_crs',
    path: '/security/crs',
    meta: { title: 'security_crs', i18nKey: 'route.security_crs', hideInMenu: true },
  },
  {
    name: 'security_exclusion',
    path: '/security/exclusion',
    meta: { title: 'security_exclusion', i18nKey: 'route.security_exclusion', hideInMenu: true },
  },
  {
    name: 'security_binding',
    path: '/security/binding',
    meta: { title: 'security_binding', i18nKey: 'route.security_binding', hideInMenu: true },
  },
  {
    name: 'security_release',
    path: '/security/release',
    meta: { title: 'security_release', i18nKey: 'route.security_release', hideInMenu: true },
  },
  {
    name: 'security_job',
    path: '/security/job',
    meta: { title: 'security_job', i18nKey: 'route.security_job', hideInMenu: true },
  },
];

// ──────────────────────────────────────────────────────────────────────────
// 内置路由名称（登录、异常页），在动态路由初始化时过滤
// ──────────────────────────────────────────────────────────────────────────

const BUILTIN_ROUTE_NAMES = new Set(['login', '403', '404', '500']);

// ──────────────────────────────────────────────────────────────────────────
// Route Model Hook
// ──────────────────────────────────────────────────────────────────────────

export default function useRouteModel() {
  const [isInitConstantRoute, setIsInitConstantRoute] = useState(false);
  const [isInitAuthRoute, setIsInitAuthRoute] = useState(false);
  const [constantRoutes, setConstantRoutes] = useState<Api.Route.MenuRoute[]>([]);
  const [authRoutes, setAuthRoutes] = useState<Api.Route.MenuRoute[]>([]);
  const [menus, setMenus] = useState<MenuItem[]>([]);
  const [routeHome, setRouteHome] = useState<string>(ROUTE_HOME);

  // ──────────────────────────────────────────────────────────────────────
  // 内部工具函数
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 过滤内置路由（login、403、404、500）。
   * 这些路由由前端静态注册，不应出现在动态路由列表中。
   */
  function filterBuiltinRoutes(routes: Api.Route.MenuRoute[]): Api.Route.MenuRoute[] {
    return routes.filter((route) => {
      if (route.name && BUILTIN_ROUTE_NAMES.has(route.name)) {
        return false;
      }
      return true;
    });
  }

  /**
   * 按 `meta.order` 对路由数组排序（升序，未设置 order 的视为 0）。
   * 返回新数组，不修改原数组。
   */
  function sortRoutesByOrder(routes: Api.Route.MenuRoute[]): Api.Route.MenuRoute[] {
    return [...routes].sort((a, b) => {
      const orderA = a.meta?.order ?? 0;
      const orderB = b.meta?.order ?? 0;
      return orderA - orderB;
    });
  }

  // ──────────────────────────────────────────────────────────────────────
  // 安全路由规范化
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 规范化安全路由：为 `/security` 父路由补全缺失的子路由模板并添加默认重定向。
   *
   * 遍历路由树，找到 path 为 `/security` 或以 `/security/` 开头的路由，
   * 将 `securityChildRouteTemplates` 中尚未存在的子路由追加进去，并确保父路由
   * 具有指向第一个子路由的默认重定向。
   */
  const normalizeSecurityRoutes = useCallback(
    (routes: Api.Route.MenuRoute[]): Api.Route.MenuRoute[] => {
      return routes.map((route) => {
        const isSecurityRoute =
          route.path === '/security' || route.path.startsWith('/security/');

        if (!isSecurityRoute) {
          // 递归处理嵌套子路由
          if (route.children?.length) {
            return { ...route, children: normalizeSecurityRoutes(route.children) };
          }
          return route;
        }

        // 深拷贝当前路由的 children 以避免修改原数组
        const children = route.children ? [...route.children] : [];
        const existingChildNames = new Set(children.map((c) => c.name));

        // 补全缺失的子路由模板
        for (const template of securityChildRouteTemplates) {
          if (!existingChildNames.has(template.name)) {
            children.push({ ...template });
          }
        }

        // 确保父路由有默认重定向（指向第一个子路由）
        const result = { ...route, children };
        const firstChildPath = children[0]?.path;
        if (firstChildPath && !(route.meta as Record<string, unknown>)?.redirect) {
          result.meta = { ...route.meta, redirect: firstChildPath };
        }

        return result;
      });
    },
    [],
  );

  // ──────────────────────────────────────────────────────────────────────
  // 路由树到菜单树转换（Req 4.3）
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 将路由树转换为菜单树（Req 4.3）。
   *
   * - 过滤 `hideInMenu` 为 true 的路由项。
   * - 保持层级结构与 `meta.order` 排序。
   * - 从路由元数据中提取 key / label / i18nKey / icon / href。
   */
  const routeTreeToMenuTree = useCallback(
    (routes: Api.Route.MenuRoute[]): MenuItem[] => {
      const sorted = sortRoutesByOrder(routes);

      const menuItems: MenuItem[] = [];

      for (const route of sorted) {
        // 过滤隐藏菜单项
        if (route.meta?.hideInMenu) {
          continue;
        }

        const menuItem: MenuItem = {
          path: route.path,
          name: route.meta?.title ?? route.name,
        };

        if (route.meta?.i18nKey) {
          menuItem.i18nKey = route.meta.i18nKey;
        }
        if (route.meta?.icon) {
          menuItem.icon = route.meta.icon;
        }
        if (route.meta?.href) {
          menuItem.href = route.meta.href;
        }

        // 递归转换子路由
        if (route.children?.length) {
          const childMenuItems = routeTreeToMenuTree(route.children);
          if (childMenuItems.length > 0) {
            menuItem.children = childMenuItems;
          }
        }

        menuItems.push(menuItem);
      }

      return menuItems;
    },
    [],
  );

  // ──────────────────────────────────────────────────────────────────────
  // RBAC 访问控制（Req 4.5）
  // ──────────────────────────────────────────────────────────────────────

  /**
   * RBAC 访问控制判定（Req 4.5）。
   *
   * - 超级角色（`isSuper` 为 true）始终放行。
   * - 路由未配置角色限制时（`requiredRoles` 为空或未定义）放行。
   * - 否则检查用户角色集合与路由所需角色是否有交集。
   */
  const canAccess = useCallback(
    (userRoles: string[], requiredRoles?: string[], isSuper?: boolean): boolean => {
      // 超级角色始终放行
      if (isSuper) {
        return true;
      }

      // 路由未配置角色限制则放行
      if (!requiredRoles || requiredRoles.length === 0) {
        return true;
      }

      // 检查角色交集
      return requiredRoles.some((role) => userRoles.includes(role));
    },
    [],
  );

  // ──────────────────────────────────────────────────────────────────────
  // 常量路由初始化
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 初始化常量路由。
   *
   * 调用 `fetchGetConstantRoutes()` 获取常量路由，过滤内置路由（login、403、404、500），
   * 规范化安全路由子路由。失败时回退到空路由列表。
   */
  const initConstantRoute = useCallback(async () => {
    // 守卫：已初始化则跳过，防止重复请求
    if (isInitConstantRoute) return;
    try {
      const { data, error } = await fetchGetConstantRoutes();

      if (error || !data) {
        console.error('[RouteModel] Failed to fetch constant routes:', error);
        setConstantRoutes([]);
        setIsInitConstantRoute(true);
        return;
      }

      // 过滤内置路由
      const filtered = filterBuiltinRoutes(data);

      // 规范化安全路由
      const normalized = normalizeSecurityRoutes(filtered);

      setConstantRoutes(normalized);
      setIsInitConstantRoute(true);
    } catch (err) {
      console.error('[RouteModel] Failed to init constant routes:', err);
      setConstantRoutes([]);
      setIsInitConstantRoute(true);
    }
  }, [isInitConstantRoute, normalizeSecurityRoutes]);

  // ──────────────────────────────────────────────────────────────────────
  // 动态路由初始化
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 初始化用户动态路由。
   *
   * 调用 `fetchGetUserRoutes()` 获取当前用户的动态路由，规范化安全路由后
   * 生成菜单树。失败时清空路由与菜单状态。
   */
  const initAuthRoute = useCallback(async () => {
    // 守卫：已初始化则跳过，防止重复请求
    if (isInitAuthRoute) return;
    try {
      const { data, error } = await fetchGetUserRoutes();

      if (error || !data) {
        console.error('[RouteModel] Failed to fetch user routes:', error);
        setAuthRoutes([]);
        setMenus([]);
        setIsInitAuthRoute(true);
        return;
      }

      // 规范化安全路由
      const normalized = normalizeSecurityRoutes(data.routes || []);

      // 设置动态路由
      setAuthRoutes(normalized);

      // 合并常量路由与动态路由生成菜单树（Req 4.3）
      // 常量路由（如 dashboard）排在前面，动态路由排在后面
      const allRoutesForMenu = [...constantRoutes, ...normalized];
      const menuTree = routeTreeToMenuTree(allRoutesForMenu);
      setMenus(menuTree);

      // 设置首页路由（Req 4.8）
      if (data.home) {
        setRouteHome(data.home);
      }

      setIsInitAuthRoute(true);
    } catch (err) {
      console.error('[RouteModel] Failed to init auth routes:', err);
      setAuthRoutes([]);
      setMenus([]);
      setIsInitAuthRoute(true);
    }
  }, [isInitAuthRoute, constantRoutes, normalizeSecurityRoutes, routeTreeToMenuTree]);

  return {
    // 状态
    isInitConstantRoute,
    isInitAuthRoute,
    constantRoutes,
    authRoutes,
    menus,
    routeHome,

    // 方法
    initConstantRoute,
    initAuthRoute,
    canAccess,
    routeTreeToMenuTree,
    normalizeSecurityRoutes,
  };
}
