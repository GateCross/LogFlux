import { Button, Result } from 'antd';
import { history } from '@umijs/max';

/** 500 异常页占位。完整文案/国际化见任务 9.3。 */
export default function Page500() {
  return (
    <Result
      status="500"
      title="500"
      subTitle="抱歉，服务器出现错误。"
      extra={
        <Button type="primary" onClick={() => history.push('/dashboard')}>
          返回首页
        </Button>
      }
    />
  );
}
