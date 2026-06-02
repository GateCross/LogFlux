/**
 * Global search command palette (Ctrl+K / Cmd+K).
 *
 * Supports both controlled mode (open/onClose from parent) and
 * uncontrolled mode (self-managed with Ctrl+K shortcut).
 */
import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { Modal, Input, List, Tag, Empty, Typography } from 'antd';
import { SearchOutlined, EnterOutlined } from '@ant-design/icons';
import { useNavigate, useModel, useIntl } from '@umijs/max';
import type { MenuItem } from '@/models/route';

const { Text } = Typography;

/** Recursively flatten menu tree into a flat list of leaf items. */
function flattenMenu(items: MenuItem[], parentPath = ''): { path: string; name: string; i18nKey?: string; parent?: string }[] {
  const result: { path: string; name: string; i18nKey?: string; parent?: string }[] = [];
  for (const item of items) {
    if (item.hideInMenu) continue;
    if (item.children?.length) {
      result.push(...flattenMenu(item.children, item.name));
    } else if (item.path) {
      result.push({
        path: item.path,
        name: item.name,
        i18nKey: item.i18nKey,
        parent: parentPath || undefined,
      });
    }
  }
  return result;
}

interface SearchCommandProps {
  /** Controlled mode: externally managed open state */
  open?: boolean;
  /** Controlled mode: callback when the panel should close */
  onClose?: () => void;
}

export default function SearchCommand({ open: controlledOpen, onClose }: SearchCommandProps) {
  const isControlled = controlledOpen !== undefined;
  const [internalOpen, setInternalOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<any>(null);
  const navigate = useNavigate();
  const { menus } = useModel('route');
  const intl = useIntl();

  const open = isControlled ? controlledOpen : internalOpen;

  const handleClose = useCallback(() => {
    if (isControlled) {
      onClose?.();
    } else {
      setInternalOpen(false);
    }
    setQuery('');
  }, [isControlled, onClose]);

  // Flatten all menu items
  const allItems = useMemo(() => flattenMenu(menus || []), [menus]);

  // Filter matching items (fuzzy match name and path)
  const filteredItems = useMemo(() => {
    if (!query.trim()) return allItems.slice(0, 20);
    const q = query.toLowerCase();
    return allItems
      .filter((item) => item.name.toLowerCase().includes(q) || item.path.toLowerCase().includes(q))
      .slice(0, 20);
  }, [query, allItems]);

  // Reset active index on query change
  useEffect(() => { setActiveIndex(0); }, [query]);

  // Navigate to selected item
  const handleSelect = useCallback((item: { path: string }) => {
    handleClose();
    navigate(item.path);
  }, [navigate, handleClose]);

  // Global keyboard shortcut (Ctrl+K / Cmd+K) — only in uncontrolled mode
  useEffect(() => {
    if (isControlled) return; // parent manages Ctrl+K in controlled mode
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setInternalOpen((prev) => !prev);
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [isControlled]);

  // In-panel keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActiveIndex((prev) => Math.min(prev + 1, filteredItems.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActiveIndex((prev) => Math.max(prev - 1, 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (filteredItems[activeIndex]) {
        handleSelect(filteredItems[activeIndex]);
      }
    }
  }, [filteredItems, activeIndex, handleSelect]);

  const t = useCallback((id: string, fb: string) => intl.formatMessage({ id, defaultMessage: fb }), [intl]);

  return (
    <Modal
      open={open}
      onCancel={handleClose}
      footer={null}
      closable={false}
      width={520}
      styles={{
        body: { padding: 0 },
        mask: { backgroundColor: 'rgba(0, 0, 0, 0.45)' },
      }}
      centered
    >
      <div style={{ padding: '16px 20px 0' }}>
        <Input
          ref={inputRef}
          size="large"
          prefix={<SearchOutlined />}
          placeholder={t('search.placeholder', 'Search pages...')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          autoFocus
          allowClear
        />
      </div>

      <div style={{ maxHeight: 400, overflowY: 'auto', padding: '8px 0' }}>
        {filteredItems.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={t('search.noResults', 'No matching pages found')}
            style={{ padding: '24px 0' }}
          />
        ) : (
          <List
            dataSource={filteredItems}
            renderItem={(item, index) => (
              <List.Item
                onClick={() => handleSelect(item)}
                style={{
                  padding: '10px 20px',
                  cursor: 'pointer',
                  backgroundColor: index === activeIndex ? 'var(--ant-color-primary-bg)' : 'transparent',
                  borderLeft: index === activeIndex ? '3px solid var(--ant-color-primary)' : '3px solid transparent',
                }}
                onMouseEnter={() => setActiveIndex(index)}
              >
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
                  <div>
                    <Text strong={index === activeIndex}>
                      {item.i18nKey ? t(item.i18nKey, item.name) : item.name}
                    </Text>
                    {item.parent && (
                      <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
                        {item.parent}
                      </Text>
                    )}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Tag style={{ margin: 0 }}>{item.path}</Tag>
                    {index === activeIndex && <EnterOutlined style={{ color: 'var(--ant-color-primary)' }} />}
                  </div>
                </div>
              </List.Item>
            )}
          />
        )}
      </div>

      <div style={{ padding: '8px 20px', borderTop: '1px solid var(--ant-color-border)', display: 'flex', justifyContent: 'space-between', fontSize: 12, color: 'var(--ant-color-text-secondary)' }}>
        <span>
          <kbd>↑↓</kbd> {t('search.navigate', 'Navigate')} <kbd>↵</kbd> {t('search.select', 'Select')} <kbd>Esc</kbd> {t('search.close', 'Close')}
        </span>
        <span>{filteredItems.length} {t('search.results', 'results')}</span>
      </div>
    </Modal>
  );
}
