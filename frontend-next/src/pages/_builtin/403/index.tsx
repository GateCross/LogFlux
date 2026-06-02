/**
 * 403 无权限页（任务 9.3），支持 i18n。
 */
import { Button, Result } from 'antd';
import { history, useIntl } from '@umijs/max';
import { ROUTE_HOME } from '@/constants/app';

export default function Page403() {
  const intl = useIntl();

  return (
    <Result
      status="403"
      title="403"
      subTitle={intl.formatMessage({
        id: 'page.403.title',
        defaultMessage: '抱歉，您没有权限访问此页面。',
      })}
      extra={
        <Button type="primary" onClick={() => history.push(`/${ROUTE_HOME}`)}>
          {intl.formatMessage({
            id: 'common.backToHome',
            defaultMessage: '返回首页',
          })}
        </Button>
      }
    />
  );
}
