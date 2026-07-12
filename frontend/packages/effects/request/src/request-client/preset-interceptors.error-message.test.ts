import { describe, expect, it, vi } from 'vitest';

import { errorMessageResponseInterceptor } from './preset-interceptors';

describe('errorMessageResponseInterceptor (errorMessageMode suppress)', () => {
  it('does not call makeErrorMessage when errorMessageMode is none on error.config', async () => {
    const makeErrorMessage = vi.fn();
    const interceptor = errorMessageResponseInterceptor(makeErrorMessage);
    const error = {
      config: { errorMessageMode: 'none' as const },
      response: { status: 500, data: { message: 'boom' } },
      message: 'Request failed',
    };

    await expect(interceptor.rejected?.(error)).rejects.toBe(error);
    expect(makeErrorMessage).not.toHaveBeenCalled();
  });

  it('does not call makeErrorMessage when errorMessageMode is none on nested response.config', async () => {
    const makeErrorMessage = vi.fn();
    const interceptor = errorMessageResponseInterceptor(makeErrorMessage);
    // defaultResponseInterceptor 业务码失败 throw 的形态：{ config, response, ... }
    const error = {
      config: { errorMessageMode: 'none' as const, responseReturn: 'data' },
      data: { code: 500, message: '业务失败' },
      response: {
        status: 200,
        data: { code: 500, message: '业务失败' },
        config: { errorMessageMode: 'none' as const },
      },
    };

    await expect(interceptor.rejected?.(error)).rejects.toBe(error);
    expect(makeErrorMessage).not.toHaveBeenCalled();
  });

  it('calls makeErrorMessage at most once when errorMessageMode is not none', async () => {
    const makeErrorMessage = vi.fn();
    const interceptor = errorMessageResponseInterceptor(makeErrorMessage);
    const error = {
      config: { errorMessageMode: 'message' as const },
      response: { status: 500, data: { message: 'server' } },
      message: 'Request failed with status code 500',
    };

    await expect(interceptor.rejected?.(error)).rejects.toBe(error);
    expect(makeErrorMessage).toHaveBeenCalledTimes(1);
  });

  it('calls makeErrorMessage once when errorMessageMode is unset (default toast)', async () => {
    const makeErrorMessage = vi.fn();
    const interceptor = errorMessageResponseInterceptor(makeErrorMessage);
    const error = {
      config: {},
      response: { status: 404, data: {} },
      message: 'Not Found',
    };

    await expect(interceptor.rejected?.(error)).rejects.toBe(error);
    expect(makeErrorMessage).toHaveBeenCalledTimes(1);
  });
});
