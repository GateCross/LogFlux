/**
 * 用户中心页面（任务 19.1 / Req 14.1–14.6）。
 *
 * 展示当前用户名与角色集合；修改偏好调用 `/api/user/preferences` 持久化，
 * 成功后同步本地偏好；失败展示消息并保留修改前本地偏好。
 */
import { useState, useCallback } from 'react';
import {
  Card, Tabs, Descriptions, Tag, Form, Input, Select, Button,
  message, Space, Alert, ColorPicker, Switch, Row, Col,
} from 'antd';
import { PageContainer } from '@ant-design/pro-components';
import { useModel, useIntl } from '@umijs/max';
import useAuthModel from '@/models/auth';
import useAppModel from '@/models/app';
import type { UserPreferences, ThemeScheme, LangType, LayoutMode } from '@/utils/preferences';
import {
  THEME_SCHEMES, LANGS, LAYOUT_MODES,
  createDefaultPreferences, isValidThemeColor,
} from '@/utils/preferences';

export default function UserCenterPage() {
  const intl = useIntl();
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const { userInfo } = useModel('auth' as any) as unknown as ReturnType<typeof useAuthModel>;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const { preferences, updatePreferences } = useModel('app' as any) as unknown as ReturnType<typeof useAppModel>;

  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [colorError, setColorError] = useState('');

  // Sync form with current preferences
  useState(() => {
    form.setFieldsValue({
      themeScheme: preferences.theme.themeScheme,
      themeColor: preferences.theme.themeColor,
      lang: preferences.lang,
      layoutMode: preferences.layout.mode,
      siderCollapse: preferences.layout.siderCollapse,
      watermarkVisible: preferences.watermark.visible,
      watermarkText: preferences.watermark.text || '',
    });
  });

  const handleSave = useCallback(async () => {
    const values = form.getFieldsValue();

    // Validate theme color (Req 6.3)
    const themeColor = values.themeColor || '';
    if (themeColor && !isValidThemeColor(themeColor)) {
      setColorError('主色值不合法，请输入合法的 hex 或 rgb 颜色');
      return;
    }
    setColorError('');

    const newPrefs: UserPreferences = {
      ...preferences,
      theme: {
        ...preferences.theme,
        themeScheme: values.themeScheme as ThemeScheme,
        themeColor: themeColor || preferences.theme.themeColor,
      },
      lang: values.lang as LangType,
      layout: {
        mode: values.layoutMode as LayoutMode,
        siderCollapse: values.siderCollapse ?? preferences.layout.siderCollapse,
      },
      watermark: {
        visible: values.watermarkVisible ?? preferences.watermark.visible,
        text: values.watermarkText || undefined,
      },
    };

    setSaving(true);
    try {
      const success = await updatePreferences(newPrefs);
      if (success) {
        message.success(intl.formatMessage({ id: 'common.saveSuccess', defaultMessage: '保存成功' }));
      }
      // Failure is handled in updatePreferences (rollback + error message)
    } finally {
      setSaving(false);
    }
  }, [form, preferences, updatePreferences, intl]);

  return (
    <PageContainer title={intl.formatMessage({ id: 'route.user_center', defaultMessage: '用户中心' })}>
      <Tabs
        defaultActiveKey="profile"
        items={[
          {
            key: 'profile',
            label: intl.formatMessage({ id: 'user.profile', defaultMessage: '个人信息' }),
            children: (
              <Card>
                <Descriptions column={2}>
                  <Descriptions.Item label={intl.formatMessage({ id: 'user.username', defaultMessage: '用户名' })}>
                    {userInfo.username || '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'user.roles', defaultMessage: '角色' })}>
                    <Space>
                      {userInfo.roles.length > 0
                        ? userInfo.roles.map(role => <Tag key={role} color="blue">{role}</Tag>)
                        : '-'}
                    </Space>
                  </Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'user.userId', defaultMessage: '用户ID' })}>
                    {userInfo.userId || '-'}
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            ),
          },
          {
            key: 'preferences',
            label: intl.formatMessage({ id: 'user.preferences', defaultMessage: '偏好设置' }),
            children: (
              <Card>
                <Form form={form} layout="vertical">
                  <Row gutter={24}>
                    <Col span={12}>
                      <Form.Item
                        name="themeScheme"
                        label={intl.formatMessage({ id: 'theme.scheme', defaultMessage: '主题方案' })}
                      >
                        <Select
                          options={THEME_SCHEMES.map(s => ({
                            label: s === 'light' ? '明亮' : s === 'dark' ? '暗黑' : '跟随系统',
                            value: s,
                          }))}
                        />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name="themeColor"
                        label={intl.formatMessage({ id: 'theme.color', defaultMessage: '主题主色' })}
                      >
                        <Input placeholder="#646cff" />
                      </Form.Item>
                      {colorError && <Alert type="error" message={colorError} style={{ marginBottom: 16 }} />}
                    </Col>
                  </Row>
                  <Row gutter={24}>
                    <Col span={12}>
                      <Form.Item
                        name="lang"
                        label={intl.formatMessage({ id: 'theme.lang', defaultMessage: '语言' })}
                      >
                        <Select
                          options={[
                            { label: '简体中文', value: 'zh-CN' },
                            { label: 'English', value: 'en-US' },
                          ]}
                        />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name="layoutMode"
                        label={intl.formatMessage({ id: 'theme.layoutMode', defaultMessage: '布局模式' })}
                      >
                        <Select
                          options={LAYOUT_MODES.map(m => ({ label: m, value: m }))}
                        />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Row gutter={24}>
                    <Col span={12}>
                      <Form.Item name="watermarkVisible" label="水印" valuePropName="checked">
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item name="watermarkText" label="水印文案">
                        <Input placeholder="LogFlux" maxLength={64} />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Form.Item>
                    <Button type="primary" loading={saving} onClick={handleSave}>
                      {intl.formatMessage({ id: 'common.confirm', defaultMessage: '保存' })}
                    </Button>
                  </Form.Item>
                </Form>
              </Card>
            ),
          },
        ]}
      />
    </PageContainer>
  );
}
