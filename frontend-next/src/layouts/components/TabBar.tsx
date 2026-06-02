/**
 * Tab bar component for multi-tab navigation.
 *
 * Supports: open, switch, close, middle-click close, right-click context menu, pin/unpin.
 */
import { Tag } from 'antd';
import { useLocation, useNavigate, useModel, useIntl } from '@umijs/max';
import { theme as antdTheme } from 'antd';
import { useCallback, useRef, useEffect, useState } from 'react';

export default function TabBar() {
  const navigate = useNavigate();
  const location = useLocation();
  const intl = useIntl();
  const { token } = antdTheme.useToken();

  const tabModel = useModel('tab');
  const tabs = tabModel?.tabs ?? [];
  const activeTabId = tabModel?.activeTabId;
  const switchTab = tabModel?.switchTab;
  const openTab = tabModel?.openTab;
  const closeTab = tabModel?.closeTab;
  const cacheTabs = tabModel?.cacheTabs;

  const [isDark, setIsDark] = useState(false);
  const skipSyncRef = useRef(false);

  // Skip the location→tab sync on the very first render so that `initHomeTab`
  // (called by the parent layout's lifecycle effect) can establish the initial
  // tab state without interference. Without this, the child-first effect ordering
  // causes TabBar to create a tab with id='/dashboard' while initHomeTab creates
  // one with id='dashboard' — resulting in two duplicate home tabs.
  const isFirstMount = useRef(true);

  // Sync theme state for background color
  useEffect(() => {
    const mql = window.matchMedia?.('(prefers-color-scheme: dark)');
    if (!mql) return;
    const update = () => setIsDark(mql.matches);
    update();
    mql.addEventListener('change', update);
    return () => mql.removeEventListener('change', update);
  }, []);

  // Cache tabs on change
  useEffect(() => { cacheTabs?.(); }, [tabs, activeTabId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Location ↔ Tab sync
  useEffect(() => {
    // Skip on first mount — let initHomeTab establish the initial tab state.
    if (isFirstMount.current) { isFirstMount.current = false; return; }
    if (skipSyncRef.current) { skipSyncRef.current = false; return; }
    const p = location.pathname;
    const existing = tabs.find((t: any) => t.path === p);
    if (existing) { if (activeTabId !== existing.id) switchTab?.(existing.id); }
    else if (p && p !== '/login') openTab?.({ id: p, label: p.split('/').filter(Boolean).pop() || 'Home', path: p });
  }, [location.pathname]); // eslint-disable-line react-hooks/exhaustive-deps

  const onTabClick = useCallback((id: string) => {
    const tab = tabs.find((t: any) => t.id === id);
    if (!tab) return;
    skipSyncRef.current = true;
    switchTab?.(id);
    if (tab.path !== location.pathname) navigate(tab.path);
  }, [tabs, switchTab, navigate, location.pathname]);

  // Context menu state — lifted to parent via custom event
  const onTabCtx = useCallback((e: React.MouseEvent, id: string) => {
    e.preventDefault();
    e.stopPropagation();
    window.dispatchEvent(new CustomEvent('tab-context-menu', {
      detail: { tabId: id, x: e.clientX, y: e.clientY },
    }));
  }, []);

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      padding: '4px 16px',
      gap: 4,
      background: isDark ? token.colorBgContainer : '#fafafa',
      borderBottom: `1px solid ${token.colorBorderSecondary}`,
      overflowX: 'auto',
      whiteSpace: 'nowrap',
      minHeight: 38,
    }}>
      {tabs.map((tab: any) => {
        const active = tab.id === activeTabId;
        return (
          <Tag
            key={tab.id}
            closable={!tab.fixed}
            onClose={(e) => { e.preventDefault(); e.stopPropagation(); closeTab?.(tab.id); }}
            onClick={() => onTabClick(tab.id)}
            onAuxClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              if (!tab.fixed) closeTab?.(tab.id);
            }}
            onContextMenu={(e) => onTabCtx(e, tab.id)}
            style={{
              cursor: 'pointer',
              margin: 0,
              borderRadius: token.borderRadiusSM,
              background: active ? token.colorPrimaryBg : 'transparent',
              borderColor: active ? token.colorPrimaryBorder : token.colorBorder,
              color: active ? token.colorPrimaryText : token.colorText,
              fontWeight: active ? 500 : 400,
              display: 'inline-flex',
              alignItems: 'center',
              padding: '2px 8px',
              lineHeight: '22px',
            }}
          >
            {tab.fixed && <span style={{ marginRight: 4, fontSize: 10 }}>&#x1F4CC;</span>}
            {tab.i18nKey ? intl.formatMessage({ id: tab.i18nKey, defaultMessage: tab.label }) : tab.label}
          </Tag>
        );
      })}
    </div>
  );
}
