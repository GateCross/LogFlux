/**
 * Umi Max runtime configuration (app.tsx) - Task 7.4.
 *
 * Runtime exports assembled here:
 *  - `locale`           : I18n_Module integration (Task 5.1) -- reads initial language from
 *                         Preferences_Store, syncs dayjs on language change.
 *  - `getInitialState`  : Restores user info from token (Req 3.2) and loads constant routes
 *                         so that the initialState plugin exposes data to the whole app.
 *  - `patchClientRoutes`: After dynamic routes are resolved, injects auth routes into the
 *                         layout shell's children (design.md "Route_Module" dynamic injection).
 *  - `onRouteChange`    : Lightweight route guard (Req 2.4 / 3.10) -- redirects unauthenticated
 *                         users to /login and authenticated users away from /login.
 *
 * Note: The `layout` runtime config is intentionally NOT exported because the Umi built-in
 * layout plugin is disabled (layout: false in config.ts). The project uses a custom
 * ProLayout shell in `layouts/index.tsx` instead.
 *
 * Design notes:
 *  - `getInitialState` runs **before** models are available, so all data is fetched via direct
 *    service / utility imports rather than `useModel`.
 *  - Auth routes are fetched in `getInitialState` (so they are ready when `patchClientRoutes`
 *    fires) and also exposed via `initialState.authRoutes` for downstream consumers.
 *  - A module-level variable (`_cachedAuthRoutes`) bridges `getInitialState` and
 *    `patchClientRoutes` because Umi does not pass `initialState` to `patchClientRoutes`.
 *  - The ProLayout UI is rendered entirely by the custom `layouts/index.tsx` component.
 *    The Umi built-in layout plugin is disabled (layout: false) to avoid double wrapping.
 */
import type { RuntimeConfig } from '@umijs/max';
import { history } from '@umijs/max';
import { getInitialLang, setupI18n } from '@/locales';
import { getToken } from '@/models/auth';
import { fetchGetUserInfo } from '@/services/auth';
import { fetchGetConstantRoutes, fetchGetUserRoutes } from '@/services/route';
import { ROUTE_HOME } from '@/constants/app';

// ---------------------------------------------------------------------------
// I18n bootstrap (Task 5.1 -- dayjs first-screen sync + language-change subscription)
// ---------------------------------------------------------------------------

setupI18n();

/**
 * Locale runtime extension (Req 5.3 / 5.6).
 *
 * Overrides Umi's default `getLocale` (which reads `localStorage.umi_locale` / browser
 * language) so that the initial language comes from Preferences_Store instead.
 */
export const locale: RuntimeConfig['locale'] = {
  getLocale() {
    return getInitialLang();
  },
};

// ---------------------------------------------------------------------------
// Module-level bridge: getInitialState -> patchClientRoutes
// ---------------------------------------------------------------------------

/**
 * Cached auth routes populated in `getInitialState` and consumed by `patchClientRoutes`.
 *
 * Umi does not pass `initialState` to `patchClientRoutes`, so we use this module-level
 * variable to share the fetched auth routes between the two runtime hooks.
 */
let _cachedAuthRoutes: Api.Route.MenuRoute[] | undefined;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Public pages that do not require authentication. */
const PUBLIC_PATHS = new Set(['/login', '/403', '/404', '/500']);

/**
 * 将后端 component 标识符解析为 Umi 页面文件路径。
 *
 * 后端 component 有三种格式：
 *  - `"layout.base"`              → 父路由，无页面组件（不设置 component）
 *  - `"view.caddy_config"`        → 叶子路由，映射到 `@/pages/caddy/config`
 *  - `"layout.base$view.dashboard"` → 单级路由，提取 view 部分映射
 *
 * view 名称中的下划线 `_` 对应目录分隔符（如 `caddy_config` → `caddy/config`）。
 */
function resolveComponentPath(component: string): string | undefined {
  // Pattern 3: layout + view combined (single-level routes)
  if (component.includes('$view.')) {
    const viewPart = component.split('$view.')[1];
    if (viewPart) {
      return `@/pages/${viewPart.replace(/_/g, '/')}`;
    }
    return undefined;
  }

  // Pattern 2: view-only (leaf routes)
  if (component.startsWith('view.')) {
    const viewName = component.slice(5); // remove "view."
    return `@/pages/${viewName.replace(/_/g, '/')}`;
  }

  // Pattern 1: layout-only (parent routes) — no page component
  if (component.startsWith('layout.')) {
    return undefined;
  }

  // Fallback: try direct mapping
  return `@/pages/${component.replace(/_/g, '/')}`;
}

/**
 * Convert backend `Api.Route.MenuRoute[]` into Umi client-side route config objects.
 */
function menuRoutesToUmiRoutes(routes: Api.Route.MenuRoute[]): Record<string, any>[] {
  return routes.map((route) => {
    const umiRoute: Record<string, any> = {
      path: route.path,
      name: route.name,
    };

    // Map component identifier to a page file path when present.
    if (route.component) {
      const componentPath = resolveComponentPath(route.component);
      if (componentPath) {
        umiRoute.component = componentPath;
      }
    }

    // Recursively convert nested children.
    if (route.children?.length) {
      umiRoute.routes = menuRoutesToUmiRoutes(route.children);
    }

    return umiRoute;
  });
}

// ---------------------------------------------------------------------------
// getInitialState (Req 3.2 / design.md Route_Module)
// ---------------------------------------------------------------------------

/**
 * Fetch user info and constant routes before the app renders.
 *
 * Runs before Umi's model layer is initialised, so we call services directly.
 * The returned object is available to every component via `useModel('@@initialState')`.
 */
export async function getInitialState(): Promise<{
  currentUser?: Api.Auth.UserInfo;
  settings?: Record<string, any>;
  authRoutes?: Api.Route.MenuRoute[];
  constantRoutes?: Api.Route.MenuRoute[];
}> {
  const token = getToken();
  let currentUser: Api.Auth.UserInfo | undefined;
  let authRoutes: Api.Route.MenuRoute[] | undefined;
  let constantRoutes: Api.Route.MenuRoute[] | undefined;

  // Parallel: user info (when token exists) + constant routes (always needed).
  const [userInfoResult, constantRoutesResult] = await Promise.all([
    token ? fetchGetUserInfo() : Promise.resolve(null),
    fetchGetConstantRoutes(),
  ]);

  // -- User info (Req 3.2) -------------------------------------------------
  if (userInfoResult && !userInfoResult.error && userInfoResult.data) {
    currentUser = userInfoResult.data;
  }

  // -- Constant routes ------------------------------------------------------
  if (constantRoutesResult && !constantRoutesResult.error && constantRoutesResult.data) {
    constantRoutes = constantRoutesResult.data;
  }

  // -- Auth routes (only when user is authenticated) ------------------------
  if (currentUser) {
    try {
      const userRoutesResult = await fetchGetUserRoutes();
      if (!userRoutesResult.error && userRoutesResult.data) {
        authRoutes = userRoutesResult.data.routes || [];
        // Cache for patchClientRoutes (module-level bridge).
        _cachedAuthRoutes = authRoutes;
      }
    } catch (err) {
      console.error('[app.tsx] Failed to fetch user routes in getInitialState:', err);
    }
  }

  return {
    currentUser,
    settings: {},
    authRoutes,
    constantRoutes,
  };
}

// ---------------------------------------------------------------------------
// patchClientRoutes (design.md "dynamic route injection")
// ---------------------------------------------------------------------------

/**
 * Inject authenticated dynamic routes into the layout shell's children.
 *
 * Umi calls this after `getInitialState` resolves. The layout shell is the route entry
 * declared in `config/routes.ts` with `path: '/'` and `component: '@/layouts/index'`.
 * We append the converted auth routes to its `routes` array so they become accessible
 * under the protected layout.
 */
export function patchClientRoutes({ routes }: { routes: any[] }): void {
  if (!_cachedAuthRoutes?.length) return;

  // Locate the layout shell (first route with path '/' that has a routes array).
  const layoutShell = routes.find(
    (r: any) => r.path === '/' && Array.isArray(r.routes),
  );
  if (!layoutShell) return;

  const umiAuthRoutes = menuRoutesToUmiRoutes(_cachedAuthRoutes);

  // Append dynamic routes after the existing placeholder children (e.g. dashboard).
  layoutShell.routes = [...layoutShell.routes, ...umiAuthRoutes];
}

// ---------------------------------------------------------------------------
// layout (RunTimeLayoutConfig)
//
// DISABLED: The Umi built-in layout plugin is turned off (layout: false in
// config/config.ts) because the project uses a custom ProLayout shell in
// src/layouts/index.tsx.  Exporting a `layout` runtime config when the plugin
// is disabled causes "register failed, invalid key layout from plugin app.tsx".
//
// Keep the code below as reference in case the layout plugin is re-enabled.
// ---------------------------------------------------------------------------

// export const layout: RunTimeLayoutConfig = (initData) => {
//   const initialState = (initData as any).initialState as {
//     settings?: Record<string, any>;
//   } | undefined;
//
//   return {
//     // Spread any ProLayout-level settings stored in initialState.settings.
//     ...(initialState?.settings),
//   };
// };

// ---------------------------------------------------------------------------
// onRouteChange -- lightweight route guard (Req 2.4 / 3.10)
// ---------------------------------------------------------------------------

/**
 * Runs on every client-side navigation.
 *
 * Guard logic:
 *  - Unauthenticated users accessing a protected page are redirected to `/login`
 *    (with the original path preserved as a `redirect` query parameter, Req 3.10).
 *  - Authenticated users visiting `/login` are redirected to the home page (Req 2.4).
 */
export function onRouteChange({ location }: { location: { pathname: string } }): void {
  const token = getToken();
  const { pathname } = location;
  const isLoginPage = pathname === '/login';
  const isPublicPage = PUBLIC_PATHS.has(pathname);

  // Not logged in and trying to access a protected page -> redirect to login.
  if (!token && !isPublicPage) {
    history.push(`/login?redirect=${encodeURIComponent(pathname)}`);
    return;
  }

  // Logged in and visiting the login page -> redirect to home.
  if (token && isLoginPage) {
    history.push(`/${ROUTE_HOME}`);
  }
}
