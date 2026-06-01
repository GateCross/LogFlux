/**
 * 偏好同步 2s 超时回滚单元测试（任务 3.5 / Req 6.5）。
 *
 * 验证：超时/任一失败时整体回滚到更新前偏好并提示失败。
 */
import { renderHook, act } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import useAppModel from './app';
import * as authService from '@/services/auth';
import * as errMsg from '@/utils/request/err-msg';
import { DEFAULT_PREFERENCES } from '@/utils/preferences';
import { clearStorage, getStorage } from '@/utils/storage';

vi.mock('@/services/auth', () => ({
  fetchUpdateUserPreferences: vi.fn(),
}));

vi.mock('@/utils/request/err-msg', () => ({
  showErrorMsg: vi.fn(),
}));

describe('useAppModel - Preferences Sync (Req 6.5)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    clearStorage();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
    clearStorage();
  });

  it('正常同步成功：双写完成，不回滚', async () => {
    vi.mocked(authService.fetchUpdateUserPreferences).mockResolvedValue({
      data: null,
      error: null,
    } as any);

    const { result } = renderHook(() => useAppModel());

    expect(result.current.preferences).toEqual(DEFAULT_PREFERENCES);

    const newPrefs = { ...DEFAULT_PREFERENCES, theme: { ...DEFAULT_PREFERENCES.theme, themeScheme: 'dark' as const } };

    let success = false;
    act(() => {
      result.current.updatePreferences(newPrefs).then(res => { success = res; });
    });

    // 乐观更新立即生效
    expect(result.current.preferences.theme.themeScheme).toBe('dark');

    // 推进微任务让 Promise 链执行完毕
    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(success).toBe(true);
    expect(result.current.preferences.theme.themeScheme).toBe('dark');
    expect(errMsg.showErrorMsg).not.toHaveBeenCalled();

    const saved = getStorage<string>('preferences');
    expect(saved).toContain('"themeScheme":"dark"');
  });

  it('远程同步超时：2s 未响应，回滚旧偏好并提示', async () => {
    vi.mocked(authService.fetchUpdateUserPreferences).mockImplementation(() => new Promise((resolve) => {
      // 模拟永远不 resolve 或 5s 后 resolve
      setTimeout(() => resolve({ data: null, error: null } as any), 5000);
    }));

    const { result } = renderHook(() => useAppModel());

    const newPrefs = { ...DEFAULT_PREFERENCES, theme: { ...DEFAULT_PREFERENCES.theme, themeScheme: 'dark' as const } };

    let success: boolean | undefined;
    act(() => {
      result.current.updatePreferences(newPrefs).then(res => { success = res; });
    });

    // 乐观更新生效
    expect(result.current.preferences.theme.themeScheme).toBe('dark');

    // 推进 2s 触发超时
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    // 超时导致 Promise.race 抛错，进入 catch 分支，发生回滚
    expect(success).toBe(false);
    expect(result.current.preferences.theme.themeScheme).toBe('light'); // 恢复到原来的 default
    expect(errMsg.showErrorMsg).toHaveBeenCalledWith('偏好设置同步失败');

    const saved = getStorage<string>('preferences');
    expect(saved).not.toContain('"themeScheme":"dark"');
  });

  it('远程同步失败：返回 error 对象，回滚旧偏好并提示', async () => {
    vi.mocked(authService.fetchUpdateUserPreferences).mockResolvedValue({
      data: null,
      error: new Error('Network Error'),
    } as any);

    const { result } = renderHook(() => useAppModel());

    const newPrefs = { ...DEFAULT_PREFERENCES, lang: 'en-US' as const };

    let success: boolean | undefined;
    await act(async () => {
      success = await result.current.updatePreferences(newPrefs);
    });

    expect(success).toBe(false);
    expect(result.current.preferences.lang).toBe('zh-CN');
    expect(errMsg.showErrorMsg).toHaveBeenCalledWith('偏好设置同步失败');
  });
});
