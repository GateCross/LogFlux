/**
 * Umi Max runtime configuration (app.tsx).
 *
 * Runtime exports assembled here:
 *  - `locale`           : I18n_Module integration -- reads initial language from
 *                         Preferences_Store, syncs dayjs on language change.
 *  - `getInitialState`  : Restores user info from token and loads constant routes
 *                         so that the initialState plugin exposes data to the whole app.
 *  - `render`           : Compose hook that fetches auth routes and registers them
 *                         in Umi's flat route map BEFORE renderClient runs.
 *                         (patchRoutes is broken: event type without async flag.)
 *  - `layout`           : ProLayout runtime config (menuDataRender, avatar, actions, etc.).
 *  - `onRouteChange`    : Lightweight route guard -- redirects unauthenticated
 *                         users to /login and authenticated users away from /login.
 *
 * Design notes:
 *  - `render` is a compose hook (properly awaited) that runs before `renderClient`.
 *    It fetches auth routes, registers them in the flat route map, then calls `next()`.
 *  - `getInitialState` runs after `render` completes, so it reuses cached routes
 *    from `_cachedAuthRoutes` to avoid a duplicate API call.
 *  - `getInitialState` runs before models are available, so all data is fetched via direct
 *    service / utility imports rather than `useModel`.
 */
import type { RuntimeConfig } from '@umijs/max';
import React from 'react';
import { history, useIntl, useModel, useNavigate } from '@umijs/max';
import { Button, Dropdown, Space, App as AntdApp } from 'antd';
import {
  GlobalOutlined, SunOutlined, MoonOutlined,
  UserOutlined, LogoutOutlined, SearchOutlined,
  DashboardOutlined, SettingOutlined, HomeOutlined,
  FundOutlined, BarChartOutlined, PieChartOutlined,
  LineChartOutlined, AreaChartOutlined, DotChartOutlined,
  ProfileOutlined, ScheduleOutlined, ProjectOutlined,
  AlertOutlined, TagsOutlined, BranchesOutlined,
  AppstoreOutlined, MenuOutlined, FileTextOutlined,
  FormOutlined, CalendarOutlined, ClockCircleOutlined,
  LockOutlined, SafetyOutlined, AuditOutlined,
  FundProjectionScreenOutlined, OrderedListOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { getInitialLang, setupI18n, changeLang, LANG_OPTIONS } from '@/locales';
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
// Module-level bridge: render -> getInitialState
// ---------------------------------------------------------------------------

/**
 * Cached auth routes populated in the `render` hook and reused by `getInitialState`.
 *
 * The `render` hook runs before `renderClient` (and thus before `getInitialState`),
 * so it fetches auth routes and caches them here. `getInitialState` checks this
 * cache first to avoid a duplicate API call.
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
    // Reuse cached routes from patchRoutes if available (avoids duplicate API call).
    if (_cachedAuthRoutes?.length) {
      authRoutes = _cachedAuthRoutes;
    } else {
      try {
        const userRoutesResult = await fetchGetUserRoutes();
        if (!userRoutesResult.error && userRoutesResult.data) {
          authRoutes = userRoutesResult.data.routes || [];
          _cachedAuthRoutes = authRoutes;
        }
      } catch (err) {
        console.error('[app.tsx] Failed to fetch user routes in getInitialState:', err);
      }
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
// Icon mapping: backend icon identifiers → AntD icon components
// ---------------------------------------------------------------------------

const ICON_MAP: Record<string, React.ReactNode> = {
  'carbon:dashboard': <DashboardOutlined />,
  'carbon:settings': <SettingOutlined />,
  'carbon:global': <GlobalOutlined />,
  'carbon:user': <UserOutlined />,
  'carbon:home': <HomeOutlined />,
  'carbon:fund': <FundOutlined />,
  'carbon:bar-chart': <BarChartOutlined />,
  'carbon:pie-chart': <PieChartOutlined />,
  'carbon:line-chart': <LineChartOutlined />,
  'carbon:area-chart': <AreaChartOutlined />,
  'carbon:dot-chart': <DotChartOutlined />,
  'carbon:ordered-list': <OrderedListOutlined />,
  'carbon:appstore': <AppstoreOutlined />,
  'carbon:menu': <MenuOutlined />,
  'carbon:file-text': <FileTextOutlined />,
  'carbon:form': <FormOutlined />,
  'carbon:calendar': <CalendarOutlined />,
  'carbon:clock': <ClockCircleOutlined />,
  'carbon:lock': <LockOutlined />,
  'carbon:safety': <SafetyOutlined />,
  'carbon:audit': <AuditOutlined />,
  'carbon:fund-projection-screen': <FundProjectionScreenOutlined />,
  'carbon:profile': <ProfileOutlined />,
  'carbon:schedule': <ScheduleOutlined />,
  'carbon:project': <ProjectOutlined />,
  'carbon:alert': <AlertOutlined />,
  'carbon:tags': <TagsOutlined />,
  'carbon:branches': <BranchesOutlined />,
  'carbon:search': <SearchOutlined />,
  'carbon:catalog': <FileTextOutlined />,
  'carbon:view': <FundProjectionScreenOutlined />,
  'carbon:time': <ClockCircleOutlined />,
  'dashboard': <DashboardOutlined />,
  'setting': <SettingOutlined />,
  'user': <UserOutlined />,
  'home': <HomeOutlined />,
  'profile': <ProfileOutlined />,
  'alert': <AlertOutlined />,
};

function renderMenuIcon(iconStr: string | undefined): React.ReactNode {
  if (!iconStr) return undefined;
  return ICON_MAP[iconStr] || ICON_MAP[iconStr.replace(/^carbon:/, '')] || undefined;
}

// ---------------------------------------------------------------------------
// layout (RunTimeLayoutConfig) — ProLayout 原生菜单系统集成
// ---------------------------------------------------------------------------

export const layout = (initData: any) => {
  const { initialState } = initData;
  const currentUser = initialState?.currentUser as Api.Auth.UserInfo | undefined;

  // Hooks are safe here — the layout plugin renders this as a React component
  // eslint-disable-next-line react-hooks/rules-of-hooks
  const navigate = useNavigate();
  // eslint-disable-next-line react-hooks/rules-of-hooks
  const intl = useIntl();
  // eslint-disable-next-line react-hooks/rules-of-hooks
  const { modal } = AntdApp.useApp();
  // eslint-disable-next-line react-hooks/rules-of-hooks
  const authModel = useModel('auth');
  // eslint-disable-next-line react-hooks/rules-of-hooks
  const themeModel = useModel('theme');
  // eslint-disable-next-line react-hooks/rules-of-hooks
  const routeModel = useModel('route');

  const userInfo = authModel?.userInfo ?? currentUser ?? { username: '', userId: '', roles: [] };
  const logout = authModel?.logout;
  const isDark = themeModel?.currentScheme === 'dark';
  const toggleThemeScheme = themeModel?.toggleThemeScheme;
  const menus = routeModel?.menus ?? [];

  const t = (id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb });

  const langItems: MenuProps['items'] = LANG_OPTIONS.map((o) => ({
    key: o.value,
    label: o.label,
    onClick: () => changeLang(o.value),
  }));

  const userItems: MenuProps['items'] = [
    {
      key: 'center',
      icon: <UserOutlined />,
      label: t('common.userCenter'),
      onClick: () => navigate('/user/center'),
    },
    { type: 'divider' as const },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: t('common.logout'),
      onClick: () =>
        modal.confirm({
          title: t('common.tip'),
          content: t('common.logoutConfirm'),
          onOk: () => logout?.(),
        }),
    },
  ];

  // Convert route model menus (from backend) to ProLayout menu format
  const buildMenuData = (items: any[]): any[] => {
    if (!items?.length) return [];
    return items
      .filter((i: any) => !i.hideInMenu)
      .map((i: any) => {
        const children = buildMenuData(i.children);
        return {
          path: i.path,
          name: i.i18nKey ? t(i.i18nKey, i.name) : (i.name || i.path),
          icon: renderMenuIcon(i.icon),
          ...(children.length > 0 ? { routes: children } : {}),
        };
      });
  };

  return {
    navTheme: isDark ? 'realDark' : 'light',
    // Merge backend dynamic menus into ProLayout's menu data
    menuDataRender: (menuData: any[]) => {
      const dynamicMenus = buildMenuData(menus as any[]);
      if (dynamicMenus.length === 0) return menuData;
      // Deduplicate: keep dynamic menus, filter out static placeholder routes
      const dynamicPaths = new Set(dynamicMenus.map((m: any) => m.path));
      const staticMenus = menuData.filter((m: any) => !dynamicPaths.has(m.path));
      return [...staticMenus, ...dynamicMenus];
    },
    actionsRender: () => [
      <Button key="search" type="text" icon={<SearchOutlined />} />,
      <Dropdown key="lang" menu={{ items: langItems }} trigger={['click']}>
        <Button type="text" icon={<GlobalOutlined />} />
      </Dropdown>,
      <Button
        key="theme"
        type="text"
        icon={isDark ? <SunOutlined /> : <MoonOutlined />}
        onClick={() => toggleThemeScheme?.()}
      />,
    ],
    avatarProps: {
      title: userInfo.username || 'User',
      icon: <UserOutlined />,
      render: (_: any, dom: React.ReactNode) => (
        <Dropdown menu={{ items: userItems }} trigger={['click']}>
          {dom}
        </Dropdown>
      ),
    },
    headerTitleRender: () => (
      <span style={{ fontWeight: 600, fontSize: 18 }}>LogFlux</span>
    ),
    collapsedButtonRender: (_c: boolean, defaultDom: React.ReactNode) => defaultDom,
  };
};

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
