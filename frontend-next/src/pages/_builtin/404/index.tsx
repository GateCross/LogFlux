import { Button, Result } from 'antd';
import { history } from '@umijs/max';

/** 404 异常页占位。完整文案/国际化见任务 9.3。 */
export default function Page404() {
  return (
    <Result
      status="404"
      title="404"
      subTitle="抱歉，您访问的页面不存在。"
      extra={
        <Button type="primary" onClick={() => history.push('/dashboard')}>
          返回首页
        </Button>
      }
    />
  );
}
