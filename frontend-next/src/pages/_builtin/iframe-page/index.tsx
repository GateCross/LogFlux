/**
 * iframe 内嵌页（任务 9.4）。
 *
 * 功能：
 *  - 从路由参数获取目标 URL 并嵌入 iframe
 *  - 加载中状态（Spin）
 *  - 加载失败 / 超时提示与重试
 */
import { useState, useRef, useEffect } from 'react';
import { Spin, Result, Button } from 'antd';
import { useParams, useIntl } from '@umijs/max';

export default function IframePage() {
  const intl = useIntl();
  const params = useParams<{ url?: string }>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const targetUrl = params.url ? decodeURIComponent(params.url) : '';

  useEffect(() => {
    if (!targetUrl) {
      setError(true);
      setLoading(false);
      return;
    }

    const iframe = iframeRef.current;
    if (!iframe) return;

    const handleLoad = () => {
      setLoading(false);
    };
    const handleError = () => {
      setError(true);
      setLoading(false);
    };

    iframe.addEventListener('load', handleLoad);
    iframe.addEventListener('error', handleError);

    // 30 秒超时
    const timeout = setTimeout(() => {
      setLoading((prev) => {
        if (prev) {
          setError(true);
          return false;
        }
        return prev;
      });
    }, 30000);

    return () => {
      iframe.removeEventListener('load', handleLoad);
      iframe.removeEventListener('error', handleError);
      clearTimeout(timeout);
    };
  }, [targetUrl]);

  const handleRetry = () => {
    setError(false);
    setLoading(true);
    // 重新触发 iframe 加载
    if (iframeRef.current) {
      // eslint-disable-next-line no-self-assign
      iframeRef.current.src = iframeRef.current.src;
    }
  };

  if (error) {
    return (
      <Result
        status="error"
        title={intl.formatMessage({ id: 'common.error', defaultMessage: '错误' })}
        subTitle={intl.formatMessage({
          id: 'iframe.loadFailed',
          defaultMessage: '页面加载失败',
        })}
        extra={
          <Button type="primary" onClick={handleRetry}>
            {intl.formatMessage({ id: 'common.retry', defaultMessage: '重试' })}
          </Button>
        }
      />
    );
  }

  return (
    <div style={{ position: 'relative', width: '100%', height: '100%' }}>
      {loading && (
        <Spin
          size="large"
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
          }}
        />
      )}
      <iframe
        ref={iframeRef}
        src={targetUrl}
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
        style={{
          width: '100%',
          height: '100%',
          border: 'none',
          visibility: loading ? 'hidden' : 'visible',
        }}
        title="iframe-page"
      />
    </div>
  );
}
