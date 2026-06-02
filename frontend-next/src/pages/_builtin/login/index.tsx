/**
 * 登录页（任务 9.1）。
 *
 * 功能：
 *  - 用户名 / 密码输入与必填校验
 *  - 调用 auth 模型 login，处理校验失败与字段级错误
 *  - 登录成功后重定向到 redirect 参数或首页
 *  - 已登录状态自动跳转
 *  - 居中卡片式布局
 */
import { useState, useEffect } from 'react';
import { Form, Input, Button, Card, Typography, theme } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { history, useModel, useIntl } from '@umijs/max';
import useAuthModel from '@/models/auth';
import { ROUTE_HOME } from '@/constants/app';

const { Title } = Typography;

export default function LoginPage() {
  const intl = useIntl();
  const { token } = theme.useToken();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const { login, isLogin } = useModel('auth' as any) as unknown as ReturnType<typeof useAuthModel>;

  // 已登录则直接跳转
  useEffect(() => {
    if (isLogin) {
      const redirect = new URLSearchParams(window.location.search).get('redirect');
      history.push(redirect || `/${ROUTE_HOME}`);
    }
  }, [isLogin]);

  const handleSubmit = async (values: { username: string; password: string }) => {
    setSubmitting(true);
    try {
      const result = await login(values.username, values.password, true);
      if (!result.success && result.validation) {
        if (result.validation.usernameError) {
          form.setFields([{ name: 'username', errors: [result.validation.usernameError] }]);
        }
        if (result.validation.passwordError) {
          form.setFields([{ name: 'password', errors: [result.validation.passwordError] }]);
        }
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: `linear-gradient(135deg, ${token.colorPrimaryBg} 0%, ${token.colorBgLayout} 100%)`,
      }}
    >
      <Card
        style={{
          width: 420,
          borderRadius: 12,
          boxShadow: token.boxShadowSecondary,
        }}
        styles={{ body: { padding: '40px 32px 24px' } }}
      >
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <Title level={2} style={{ marginBottom: 0 }}>
            {intl.formatMessage({ id: 'system.title', defaultMessage: 'LogFlux' })}
          </Title>
        </div>

        <Form
          form={form}
          onFinish={handleSubmit}
          size="large"
          autoComplete="off"
        >
          <Form.Item
            name="username"
            rules={[
              {
                required: true,
                message: intl.formatMessage({
                  id: 'login.username.required',
                  defaultMessage: '请输入用户名',
                }),
              },
            ]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder={intl.formatMessage({
                id: 'login.username.placeholder',
                defaultMessage: '用户名',
              })}
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[
              {
                required: true,
                message: intl.formatMessage({
                  id: 'login.password.required',
                  defaultMessage: '请输入密码',
                }),
              },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder={intl.formatMessage({
                id: 'login.password.placeholder',
                defaultMessage: '密码',
              })}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={submitting}
              block
            >
              {intl.formatMessage({
                id: 'common.confirm',
                defaultMessage: '登录',
              })}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
