/**
 * Header action buttons: search, language, theme toggle, user menu.
 */
import { Dropdown, Space, Button } from 'antd';
import { GlobalOutlined, SunOutlined, MoonOutlined, UserOutlined, LogoutOutlined, SearchOutlined } from '@ant-design/icons';
import { useNavigate, useModel, useIntl } from '@umijs/max';
import { App as AntdApp } from 'antd';
import { useMemo, useCallback } from 'react';
import { LANG_OPTIONS, changeLang } from '@/locales';
import type { MenuProps } from 'antd';

interface HeaderActionsProps {
  isDark: boolean;
  onSearchClick: () => void;
}

export default function HeaderActions({ isDark, onSearchClick }: HeaderActionsProps) {
  const navigate = useNavigate();
  const intl = useIntl();
  const { modal } = AntdApp.useApp();
  const t = useCallback((id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }), [intl]);

  const themeModel = useModel('theme');
  const authModel = useModel('auth');
  const toggleThemeScheme = themeModel?.toggleThemeScheme;
  const userInfo = authModel?.userInfo ?? { username: '', userId: '', roles: [] };
  const logout = authModel?.logout;

  const langItems: MenuProps['items'] = useMemo(
    () => LANG_OPTIONS.map((o) => ({ key: o.value, label: o.label, onClick: () => changeLang(o.value) })),
    [],
  );

  const userItems: MenuProps['items'] = useMemo(() => [
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
      onClick: () => modal.confirm({
        title: t('common.tip'),
        content: t('common.logoutConfirm'),
        onOk: () => logout?.(),
      }),
    },
  ], [t, navigate, logout, modal]);

  return (
    <>
      <Button key="search" type="text" icon={<SearchOutlined />} onClick={onSearchClick} />
      <Dropdown key="lang" menu={{ items: langItems }} trigger={['click']}>
        <Button type="text" icon={<GlobalOutlined />} />
      </Dropdown>
      <Button
        key="theme"
        type="text"
        icon={isDark ? <SunOutlined /> : <MoonOutlined />}
        onClick={() => toggleThemeScheme?.()}
      />
      <Dropdown key="user" menu={{ items: userItems }} trigger={['click']}>
        <Space style={{ cursor: 'pointer' }}>
          <UserOutlined />
          <span>{userInfo.username || 'User'}</span>
        </Space>
      </Dropdown>
    </>
  );
}
