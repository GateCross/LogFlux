/**
 * 404 页面不存在（任务 9.3），支持 i18n。
 */
import { Button, Result } from 'antd';
import { history, useIntl } from '@umijs/max';
import { ROUTE_HOME } from '@/constants/app';

export default function Page404() {
  const intl = useIntl();

  return (
    <Result
      status="404"
      title="404"
      subTitle={intl.formatMessage({
        id: 'page.404.title',
        defaultMessage: '抱歉，您访问的页面不存在。',
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
