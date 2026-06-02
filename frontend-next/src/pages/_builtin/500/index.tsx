/**
 * 500 服务器错误页（任务 9.3），支持 i18n。
 */
import { Button, Result } from 'antd';
import { history, useIntl } from '@umijs/max';
import { ROUTE_HOME } from '@/constants/app';

export default function Page500() {
  const intl = useIntl();

  return (
    <Result
      status="500"
      title="500"
      subTitle={intl.formatMessage({
        id: 'page.500.title',
        defaultMessage: '抱歉，服务器出错了。',
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
