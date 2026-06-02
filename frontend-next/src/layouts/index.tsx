/**
 * ProLayout Shell — 简化版。
 * 所有模型通过 useModel 统一消费，不混用直接 import。
 */
import { ProLayout } from '@ant-design/pro-components';
import { Outlet, useLocation, useNavigate, useModel } from '@umijs/max';
import { ConfigProvider, theme as antdTheme, App as AntdApp, Dropdown, Space, Button, Tag } from 'antd';
import { GlobalOutlined, SunOutlined, MoonOutlined, UserOutlined, LogoutOutlined } from '@ant-design/icons';
import { useEffect, useState, useMemo, useRef, useCallback } from 'react';
import { useIntl } from '@umijs/max';
import { LANG_OPTIONS, changeLang } from '@/locales';
import type { MenuProps } from 'antd';

// ---------------------------------------------------------------------------
// 菜单数据转换
// ---------------------------------------------------------------------------

interface MenuItem {
  path: string;
  name?: string;
  i18nKey?: string;
  icon?: string;
  children?: MenuItem[];
  hideInMenu?: boolean;
}

function toMenuData(items: MenuItem[] | undefined, t: (id: string, fb?: string) => string): any[] {
  if (!items?.length) return [];
  return items
    .filter((i) => !i.hideInMenu)
    .map((i) => {
      const children = toMenuData(i.children, t);
      return {
        path: i.path,
        name: i.i18nKey ? t(i.i18nKey, i.name) : (i.name || i.path),
        icon: i.icon && !i.icon.includes(':') ? i.icon : undefined,
        // ProLayout 用 routes 表示子菜单，不是 children
        ...(children.length > 0 ? { routes: children } : {}),
      };
    });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function ShellLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const intl = useIntl();
  const t = useCallback((id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }), [intl]);
  const { token } = antdTheme.useToken();
  const { modal } = AntdApp.useApp();

  // ── Models（全部通过 useModel 消费）────────────────────────────────────
  const routeModel = useModel('route');
  const themeModel = useModel('theme');
  const tabModel = useModel('tab');
  const authModel = useModel('auth');
  const appModel = useModel('app');

  const menus = routeModel?.menus ?? [];
  const initConstantRoute = routeModel?.initConstantRoute;
  const initAuthRoute = routeModel?.initAuthRoute;

  const themeScheme = themeModel?.themeScheme ?? 'light';
  const themeColor = themeModel?.themeColor ?? '#1677ff';
  const toggleThemeScheme = themeModel?.toggleThemeScheme;

  const tabs = tabModel?.tabs ?? [];
  const activeTabId = tabModel?.activeTabId;
  const openTab = tabModel?.openTab;
  const switchTab = tabModel?.switchTab;
  const closeTab = tabModel?.closeTab;
  const closeOtherTabs = tabModel?.closeOtherTabs;
  const closeLeftTabs = tabModel?.closeLeftTabs;
  const closeRightTabs = tabModel?.closeRightTabs;
  const closeAllTabs = tabModel?.closeAllTabs;
  const fixTab = tabModel?.fixTab;
  const unfixTab = tabModel?.unfixTab;
  const initHomeTab = tabModel?.initHomeTab;
  const cacheTabs = tabModel?.cacheTabs;

  const userInfo = authModel?.userInfo ?? { username: '', userId: '', roles: [] };
  const logout = authModel?.logout;

  const syncFromRemote = appModel?.syncFromRemote;

  // ── Remote preferences sync ─────────────────────────────────────────────
  useEffect(() => {
    const rp = (userInfo as any)?.preferences;
    if (rp && syncFromRemote) syncFromRemote(rp);
  }, [(userInfo as any)?.preferences, syncFromRemote]);

  // ── Local state ─────────────────────────────────────────────────────────
  const [collapsed, setCollapsed] = useState(false);
  const [ctxMenu, setCtxMenu] = useState<{ visible: boolean; tabId: string | null; x: number; y: number }>({
    visible: false, tabId: null, x: 0, y: 0,
  });
  const skipSyncRef = useRef(false);
  const initRef = useRef(false);

  // ── Theme ───────────────────────────────────────────────────────────────
  const [prefersDark, setPrefersDark] = useState(() =>
    typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches,
  );
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;
    const mql = window.matchMedia('(prefers-color-scheme: dark)');
    const h = (e: MediaQueryListEvent) => setPrefersDark(e.matches);
    mql.addEventListener('change', h);
    return () => mql.removeEventListener('change', h);
  }, []);

  const isDark = useMemo(() => (themeScheme === 'auto' ? prefersDark : themeScheme === 'dark'), [themeScheme, prefersDark]);
  const algorithm = useMemo(() => (isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm), [isDark]);

  // ── Lifecycle（一次性初始化）────────────────────────────────────────────
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;
    initHomeTab?.();
    initConstantRoute?.()?.then?.(() => initAuthRoute?.());
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => { cacheTabs?.(); /* eslint-disable-next-line */ }, [tabs, activeTabId]);

  // ── Location ↔ Tab sync ─────────────────────────────────────────────────
  useEffect(() => {
    if (skipSyncRef.current) { skipSyncRef.current = false; return; }
    const p = location.pathname;
    const existing = tabs.find((t: any) => t.path === p);
    if (existing) { if (activeTabId !== existing.id) switchTab?.(existing.id); }
    else if (p && p !== '/login') openTab?.({ id: p, label: p.split('/').filter(Boolean).pop() || 'Home', path: p });
  }, [location.pathname]);

  // ── Tab handlers ────────────────────────────────────────────────────────
  const onTabClick = useCallback((id: string) => {
    const tab = tabs.find((t: any) => t.id === id);
    if (!tab) return;
    skipSyncRef.current = true;
    switchTab?.(id);
    if (tab.path !== location.pathname) navigate(tab.path);
  }, [tabs, switchTab, navigate, location.pathname]);

  const onTabMidClick = useCallback((e: React.MouseEvent, id: string) => {
    e.preventDefault(); e.stopPropagation();
    const tab = tabs.find((t: any) => t.id === id);
    if (tab && !tab.fixed) closeTab?.(id);
  }, [tabs, closeTab]);

  const onTabCtx = useCallback((e: React.MouseEvent, id: string) => {
    e.preventDefault(); e.stopPropagation();
    setCtxMenu({ visible: true, tabId: id, x: e.clientX, y: e.clientY });
  }, []);

  const closeCtx = useCallback(() => setCtxMenu((p) => ({ ...p, visible: false })), []);

  useEffect(() => {
    if (!ctxMenu.visible) return;
    const h = () => closeCtx();
    document.addEventListener('click', h);
    return () => document.removeEventListener('click', h);
  }, [ctxMenu.visible, closeCtx]);

  // ── Menu data ───────────────────────────────────────────────────────────
  const menuData = useMemo(() => toMenuData(menus as MenuItem[], t), [menus, t]);

  // ── Header menus ────────────────────────────────────────────────────────
  const langItems: MenuProps['items'] = useMemo(
    () => LANG_OPTIONS.map((o) => ({ key: o.value, label: o.label, onClick: () => changeLang(o.value) })),
    [],
  );

  const userItems: MenuProps['items'] = useMemo(() => [
    { key: 'center', icon: <UserOutlined />, label: t('common.userCenter'), onClick: () => navigate('/user/center') },
    { type: 'divider' as const },
    { key: 'logout', icon: <LogoutOutlined />, label: t('common.logout'), onClick: () => modal.confirm({ title: t('common.tip'), content: t('common.logoutConfirm'), onOk: () => logout?.() }) },
  ], [t, navigate, logout, modal]);

  // ── Tab context menu ────────────────────────────────────────────────────
  const ctxTab = ctxMenu.tabId ? tabs.find((t: any) => t.id === ctxMenu.tabId) : null;
  const ctxItems: MenuProps['items'] = useMemo(() => {
    if (!ctxMenu.tabId) return [];
    const id = ctxMenu.tabId;
    const fixed = ctxTab?.fixed;
    return [
      { key: 'close', label: t('dropdown.closeCurrent'), disabled: fixed, onClick: () => { if (!fixed) closeTab?.(id); } },
      { key: 'closeOther', label: t('dropdown.closeOther'), onClick: () => closeOtherTabs?.(id) },
      { key: 'closeLeft', label: t('dropdown.closeLeft'), onClick: () => closeLeftTabs?.(id) },
      { key: 'closeRight', label: t('dropdown.closeRight'), onClick: () => closeRightTabs?.(id) },
      { key: 'closeAll', label: t('dropdown.closeAll'), onClick: () => closeAllTabs?.() },
      { type: 'divider' as const },
      { key: 'pin', label: fixed ? t('dropdown.unpin') : t('dropdown.pin'), onClick: () => { fixed ? unfixTab?.(id) : fixTab?.(id); } },
    ];
  }, [ctxMenu.tabId, ctxTab?.fixed, t, closeTab, closeOtherTabs, closeLeftTabs, closeRightTabs, closeAllTabs, fixTab, unfixTab]);

  // ── Tab bar ─────────────────────────────────────────────────────────────
  const tabBar = (
    <div style={{ display: 'flex', alignItems: 'center', padding: '4px 16px', gap: 4, background: isDark ? token.colorBgContainer : '#fafafa', borderBottom: `1px solid ${token.colorBorderSecondary}`, overflowX: 'auto', whiteSpace: 'nowrap', minHeight: 38 }}>
      {tabs.map((tab: any) => {
        const active = tab.id === activeTabId;
        return (
          <Tag
            key={tab.id}
            closable={!tab.fixed}
            onClose={(e) => { e.preventDefault(); e.stopPropagation(); closeTab?.(tab.id); }}
            onClick={() => onTabClick(tab.id)}
            onAuxClick={(e) => onTabMidClick(e, tab.id)}
            onContextMenu={(e) => onTabCtx(e, tab.id)}
            style={{ cursor: 'pointer', margin: 0, borderRadius: token.borderRadiusSM, background: active ? token.colorPrimaryBg : 'transparent', borderColor: active ? token.colorPrimaryBorder : token.colorBorder, color: active ? token.colorPrimaryText : token.colorText, fontWeight: active ? 500 : 400, display: 'inline-flex', alignItems: 'center', padding: '2px 8px', lineHeight: '22px' }}
          >
            {tab.fixed && <span style={{ marginRight: 4, fontSize: 10 }}>&#x1F4CC;</span>}
            {tab.i18nKey ? intl.formatMessage({ id: tab.i18nKey, defaultMessage: tab.label }) : tab.label}
          </Tag>
        );
      })}
    </div>
  );

  // ── Render ──────────────────────────────────────────────────────────────
  return (
    <ConfigProvider theme={{ algorithm, token: { colorPrimary: themeColor } }}>
      <AntdApp>
        <ProLayout
          title="LogFlux"
          logo={false}
          layout="mix"
          fixSiderbar
          fixedHeader
          navTheme={isDark ? 'realDark' : 'light'}
          siderWidth={220}
          collapsed={collapsed}
          onCollapse={setCollapsed}
          location={location}
          menuDataRender={() => menuData}
          collapsedButtonRender={(c) => <Button type="text" size="small" onClick={() => setCollapsed(!c)}>{c ? '▶' : '◀'}</Button>}
          headerTitleRender={() => <span style={{ fontWeight: 600, fontSize: 18 }}>LogFlux</span>}
          actionsRender={() => [
            <Dropdown key="lang" menu={{ items: langItems }} trigger={['click']}><Button type="text" icon={<GlobalOutlined />} /></Dropdown>,
            <Button key="theme" type="text" icon={isDark ? <SunOutlined /> : <MoonOutlined />} onClick={() => toggleThemeScheme?.()} />,
            <Dropdown key="user" menu={{ items: userItems }} trigger={['click']}><Space style={{ cursor: 'pointer' }}><UserOutlined /><span>{userInfo.username || 'User'}</span></Space></Dropdown>,
          ]}
          menuItemRender={(item, dom) => <div onClick={() => item.path && navigate(item.path)}>{dom}</div>}
          contentStyle={{ padding: 0 }}
        >
          {tabBar}
          <div style={{ padding: 16 }}><Outlet /></div>
        </ProLayout>

        {/* 右键菜单 */}
        {ctxMenu.visible && ctxMenu.tabId && (
          <div style={{ position: 'fixed', inset: 0, zIndex: 1050 }} onClick={closeCtx} onContextMenu={(e) => { e.preventDefault(); closeCtx(); }}>
            <div style={{ position: 'absolute', top: ctxMenu.y, left: ctxMenu.x, background: token.colorBgElevated, borderRadius: token.borderRadiusLG, boxShadow: token.boxShadowSecondary, padding: '4px 0', minWidth: 140, zIndex: 1051 }}>
              {ctxItems?.map((item: any) => (
                <div
                  key={item.key}
                  onClick={() => { item.onClick?.(); closeCtx(); }}
                  style={{ padding: '6px 16px', cursor: item.disabled ? 'not-allowed' : 'pointer', color: item.disabled ? token.colorTextDisabled : token.colorText, fontSize: token.fontSizeSM }}
                  onMouseEnter={(e) => { if (!item.disabled) (e.currentTarget as HTMLDivElement).style.background = token.colorBgTextHover; }}
                  onMouseLeave={(e) => { (e.currentTarget as HTMLDivElement).style.background = 'transparent'; }}
                >{item.label}</div>
              ))}
            </div>
          </div>
        )}
      </AntdApp>
    </ConfigProvider>
  );
}
