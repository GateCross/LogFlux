import { Button, Result } from 'antd';
import { history } from '@umijs/max';

/** 403 异常页占位。完整文案/国际化见任务 9.3。 */
export default function Page403() {
  return (
    <Result
      status="403"
      title="403"
      subTitle="抱歉，您没有权限访问此页面。"
      extra={
        <Button type="primary" onClick={() => history.push('/dashboard')}>
          返回首页
        </Button>
      }
    />
  );
}
