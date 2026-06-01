/**
 * 令牌刷新单飞与重放至多一次单元测试（任务 2.8 / Req 2.6、3.8、3.9）。
 *
 * 验证：并发请求共享单一刷新、刷新成功重放一次、刷新失败清空鉴权。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AxiosError } from 'axios';
import request, { 
  handleExpiredRequest, 
  requestState, 
  resetRequestState,
  setRefreshTokenCaller,
  resetRefreshTokenCaller,
  setRequestAuthAdapter,
  resetRequestAuthAdapter,
  instance,
  type RequestAuthAdapter
} from './index';

describe('Request Layer - Refresh Token & Retry', () => {
  let mockAuthAdapter: ReturnType<typeof vi.mocked<RequestAuthAdapter>>;
  let mockAdapter: ReturnType<typeof vi.fn>;
  let originalAdapter: any;

  beforeEach(() => {
    vi.useFakeTimers();
    resetRequestState();
    
    // 构造鉴权适配器 Mock
    mockAuthAdapter = {
      getToken: vi.fn(() => 'old-token'),
      getRefreshToken: vi.fn(() => 'old-refresh-token'),
      setToken: vi.fn(),
      setRefreshToken: vi.fn(),
      onLogout: vi.fn(),
    };
    setRequestAuthAdapter(mockAuthAdapter as unknown as RequestAuthAdapter);

    // 拦截底层的 Axios 请求通过替换 adapter
    originalAdapter = instance.defaults.adapter;
    mockAdapter = vi.fn();
    instance.defaults.adapter = mockAdapter as any;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
    resetRequestState();
    resetRefreshTokenCaller();
    resetRequestAuthAdapter();
    instance.defaults.adapter = originalAdapter;
  });

  function createMockError(status: number, config: any = {}, message: string = 'error') {
    const err = new AxiosError(message);
    err.config = config;
    err.response = { status, data: {}, statusText: '', headers: {}, config };
    return err;
  }

  function createMockResponse(data: any, config: any = {}, status: number = 200) {
    return {
      data,
      status,
      statusText: 'OK',
      headers: {},
      config,
      request: {},
    };
  }

  it('并发请求共享单一刷新单飞 promise', async () => {
    let refreshCallCount = 0;
    setRefreshTokenCaller(async () => {
      refreshCallCount++;
      await new Promise(resolve => setTimeout(resolve, 500));
      return { token: 'new-token', refreshToken: 'new-refresh-token' };
    });

    const p1 = handleExpiredRequest();
    const p2 = handleExpiredRequest();
    const p3 = handleExpiredRequest();

    expect(refreshCallCount).toBe(1);
    expect(requestState.refreshTokenPromise).not.toBeNull();

    vi.advanceTimersByTime(500);

    const [r1, r2, r3] = await Promise.all([p1, p2, p3]);

    expect(r1).toBe(true);
    expect(r2).toBe(true);
    expect(r3).toBe(true);
    expect(refreshCallCount).toBe(1);

    vi.advanceTimersByTime(1000);
    expect(requestState.refreshTokenPromise).toBeNull();
  });

  it('刷新成功：重放一次原请求，只重放一次', async () => {
    setRefreshTokenCaller(async () => ({
      token: 'new-token',
      refreshToken: 'new-refresh',
    }));

    mockAdapter.mockImplementationOnce((config) => Promise.reject(createMockError(401, config)))
      .mockImplementationOnce((config) => Promise.resolve(createMockResponse({ code: '200', data: 'success-data' }, config)));

    const resp = await request({ url: '/api/data' });

    expect(mockAuthAdapter.setToken).toHaveBeenCalledWith('new-token');
    expect(mockAuthAdapter.setRefreshToken).toHaveBeenCalledWith('new-refresh');

    expect(resp.data).toBe('success-data');
    expect(resp.error).toBeNull();

    expect(mockAdapter).toHaveBeenCalledTimes(2);
  });

  it('刷新失败：不重放并登出', async () => {
    setRefreshTokenCaller(async () => null);

    mockAdapter.mockImplementationOnce((config) => Promise.reject(createMockError(401, config)));

    const resp = await request({ url: '/api/data' });

    expect(mockAuthAdapter.setToken).not.toHaveBeenCalled();
    expect(mockAuthAdapter.onLogout).toHaveBeenCalled();
    
    expect(resp.data).toBeNull();
    expect(resp.error).toBeInstanceOf(Error);
    expect((resp.error as any).response?.status).toBe(401);

    expect(mockAdapter).toHaveBeenCalledTimes(1);
  });

  it('重放后仍返回 401：不再触发刷新，直接登出', async () => {
    setRefreshTokenCaller(async () => ({
      token: 'new-token',
      refreshToken: 'new-refresh',
    }));

    mockAdapter.mockImplementationOnce((config) => Promise.reject(createMockError(401, config, '401 first')))
      .mockImplementationOnce((config) => Promise.reject(createMockError(401, config, '401 second')));

    const resp = await request({ url: '/api/data' });

    expect(mockAuthAdapter.setToken).toHaveBeenCalledWith('new-token');
    expect(mockAuthAdapter.onLogout).toHaveBeenCalled();

    expect(resp.data).toBeNull();
    expect(resp.error?.message).toBe('401 second');
    expect(mockAdapter).toHaveBeenCalledTimes(2);
  });

  it('后端 2xx 响应携带 EXPIRED 业务码：执行刷新和重放', async () => {
    setRefreshTokenCaller(async () => ({
      token: 'new-token',
      refreshToken: 'new-refresh',
    }));

    mockAdapter.mockImplementationOnce((config) => Promise.resolve(createMockResponse({ code: '9999', message: 'Token Expired' }, config)))
      .mockImplementationOnce((config) => Promise.resolve(createMockResponse({ code: '200', data: 'retry-success' }, config)));

    const resp = await request({ url: '/api/data' });

    expect(mockAuthAdapter.setToken).toHaveBeenCalledWith('new-token');
    expect(resp.data).toBe('retry-success');
    expect(resp.error).toBeNull();
    expect(mockAdapter).toHaveBeenCalledTimes(2);
  });
});
