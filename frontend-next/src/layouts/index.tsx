/**
 * Shell Layout — content area with TabBar + lifecycle management.
 *
 * ProLayout itself is handled by the UmiJS Max layout plugin (config.ts).
 * This component only renders inside ProLayout's content area.
 */
import { Outlet, useModel } from '@umijs/max';
import { App as AntdApp, ConfigProvider } from 'antd';
import { useEffect, useState, useRef } from 'react';
import { useIntl } from '@umijs/max';
import { setModalLogoutHandler } from '@/utils/request';
import SearchCommand from '@/components/SearchCommand';
import TabBar from './components/TabBar';
import TabContextMenu from './components/TabContextMenu';
import { useThemeSetup } from './hooks/useThemeSetup';

export default function ShellLayout() {
  const intl = useIntl();
  const t = (id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb });
  const { modal } = AntdApp.useApp();

  // ── Theme ────────────────────────────────────────────────────────────────
  const { algorithm, themeColor } = useThemeSetup();

  // ── Models ───────────────────────────────────────────────────────────────
  const routeModel = useModel('route');
  const tabModel = useModel('tab');
  const appModel = useModel('app');
  const authModel = useModel('auth');
  const themeModel = useModel('theme');

  const initConstantRoute = routeModel?.initConstantRoute;
  const initAuthRoute = routeModel?.initAuthRoute;
  const initHomeTab = tabModel?.initHomeTab;

  const syncFromRemote = appModel?.syncFromRemote;
  const appPreferences = appModel?.preferences;
  const userInfo = authModel?.userInfo ?? { username: '', userId: '', roles: [] };

  const initTheme = themeModel?.initTheme;

  // Read cached auth routes from initialState
  const { initialState } = useModel('@@initialState');
  const cachedAuthRoutes = (initialState as any)?.authRoutes as Api.Route.MenuRoute[] | undefined;

  // ── Remote preferences sync ──────────────────────────────────────────────
  useEffect(() => {
    const rp = (userInfo as any)?.preferences;
    if (rp && syncFromRemote) syncFromRemote(rp);
  }, [(userInfo as any)?.preferences, syncFromRemote]);

  // ── Apply theme from preferences ────────────────────────────────────────
  useEffect(() => {
    if (appPreferences && initTheme) {
      initTheme(appPreferences);
    }
  }, [appPreferences, initTheme]);

  // ── Inject ModalLogoutHandler ────────────────────────────────────────────
  useEffect(() => {
    setModalLogoutHandler((message: string, onConfirm: () => void) => {
      modal.error({
        title: t('common.tip'),
        content: message,
        okText: t('common.confirm'),
        onOk: onConfirm,
      });
    });
  }, [modal, t]);

  // ── Lifecycle ────────────────────────────────────────────────────────────
  const initRef = useRef(false);
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;
    initHomeTab?.();
    initConstantRoute?.()?.then?.(() => initAuthRoute?.(cachedAuthRoutes));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // ── Search command ───────────────────────────────────────────────────────
  const [searchOpen, setSearchOpen] = useState(false);
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setSearchOpen((prev) => !prev);
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  // ── Render ──────────────────────────────────────────────────────────────
  return (
    <ConfigProvider theme={{ algorithm, token: { colorPrimary: themeColor } }}>
      <AntdApp>
        <TabBar />
        <div style={{ padding: 16 }}>
          <Outlet />
        </div>

        {/* Global search command palette */}
        <SearchCommand open={searchOpen} onClose={() => setSearchOpen(false)} />

        {/* Tab context menu */}
        <TabContextMenu />
      </AntdApp>
    </ConfigProvider>
  );
}
