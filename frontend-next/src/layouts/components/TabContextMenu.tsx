/**
 * Tab context menu — listens for 'tab-context-menu' custom events from TabBar.
 *
 * Provides: close, close-other, close-left, close-right, close-all, pin/unpin.
 */
import { useState, useEffect, useCallback, useMemo } from 'react';
import { theme as antdTheme } from 'antd';
import { useModel, useIntl } from '@umijs/max';

export default function TabContextMenu() {
  const { token } = antdTheme.useToken();
  const intl = useIntl();
  const t = useCallback((id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }), [intl]);

  const tabModel = useModel('tab');
  const tabs = tabModel?.tabs ?? [];
  const closeTab = tabModel?.closeTab;
  const closeOtherTabs = tabModel?.closeOtherTabs;
  const closeLeftTabs = tabModel?.closeLeftTabs;
  const closeRightTabs = tabModel?.closeRightTabs;
  const closeAllTabs = tabModel?.closeAllTabs;
  const fixTab = tabModel?.fixTab;
  const unfixTab = tabModel?.unfixTab;

  const [ctxMenu, setCtxMenu] = useState<{ visible: boolean; tabId: string | null; x: number; y: number }>({
    visible: false, tabId: null, x: 0, y: 0,
  });

  // Listen for context menu events from TabBar
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      setCtxMenu({ visible: true, tabId: detail.tabId, x: detail.x, y: detail.y });
    };
    window.addEventListener('tab-context-menu', handler);
    return () => window.removeEventListener('tab-context-menu', handler);
  }, []);

  const closeCtx = useCallback(() => setCtxMenu((p) => ({ ...p, visible: false })), []);

  // Close on click outside
  useEffect(() => {
    if (!ctxMenu.visible) return;
    const h = () => closeCtx();
    document.addEventListener('click', h);
    return () => document.removeEventListener('click', h);
  }, [ctxMenu.visible, closeCtx]);

  const ctxTab = ctxMenu.tabId ? tabs.find((t: any) => t.id === ctxMenu.tabId) : null;

  const items = useMemo(() => {
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

  if (!ctxMenu.visible || !ctxMenu.tabId) return null;

  return (
    <div
      style={{ position: 'fixed', inset: 0, zIndex: 1050 }}
      onClick={closeCtx}
      onContextMenu={(e) => { e.preventDefault(); closeCtx(); }}
    >
      <div style={{
        position: 'absolute',
        top: ctxMenu.y,
        left: ctxMenu.x,
        background: token.colorBgElevated,
        borderRadius: token.borderRadiusLG,
        boxShadow: token.boxShadowSecondary,
        padding: '4px 0',
        minWidth: 140,
        zIndex: 1051,
      }}>
        {items?.map((item: any) => (
          <div
            key={item.key}
            onClick={() => { item.onClick?.(); closeCtx(); }}
            style={{
              padding: '6px 16px',
              cursor: item.disabled ? 'not-allowed' : 'pointer',
              color: item.disabled ? token.colorTextDisabled : token.colorText,
              fontSize: token.fontSizeSM,
            }}
            onMouseEnter={(e) => {
              if (!item.disabled) (e.currentTarget as HTMLDivElement).style.background = token.colorBgTextHover;
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLDivElement).style.background = 'transparent';
            }}
          >
            {item.label}
          </div>
        ))}
      </div>
    </div>
  );
}
