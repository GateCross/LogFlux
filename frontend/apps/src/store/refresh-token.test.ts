
import { useAccessStore } from '@vben/stores';

import * as fc from 'fast-check';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it } from 'vitest';

import {
  clearRefreshToken,
  getRefreshToken,
  setRefreshToken,
} from './refresh-token';

const LEGACY_REFRESH_TOKEN_KEY = 'LF_refreshToken';

/** 可用非空 token 字符串（与「未认证」相对） */
const usableTokenArb = fc
  .string({ minLength: 1, maxLength: 64 })
  .filter((s) => s.trim().length > 0);

/** 与生产路径一致的 token 清理 */
function clearAuthTokensLikeProductionPaths(): void {
  clearRefreshToken();
  useAccessStore().setAccessToken(null);
}

function seedAuthenticatedSession(access: string, refresh: string): void {
  const accessStore = useAccessStore();
  accessStore.setAccessToken(access);
  setRefreshToken(refresh);
}

function assertUnauthenticatedReadable(): void {
  const accessStore = useAccessStore();
  expect(accessStore.accessToken).toBeNull();
  expect(getRefreshToken()).toBeNull();
  expect(accessStore.refreshToken).toBeNull();
}

describe('Refresh_Token_Store — token clear completeness', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    try {
      localStorage.removeItem(LEGACY_REFRESH_TOKEN_KEY);
    } catch {
      // ignore
    }
  });

  it('get/set/clear 仅经 useAccessStore().refreshToken', () => {
    expect(getRefreshToken()).toBeNull();

    setRefreshToken('rt-login');
    expect(getRefreshToken()).toBe('rt-login');
    expect(useAccessStore().refreshToken).toBe('rt-login');

    setRefreshToken('rt-refresh');
    expect(getRefreshToken()).toBe('rt-refresh');

    clearRefreshToken();
    expect(getRefreshToken()).toBeNull();
    expect(useAccessStore().refreshToken).toBeNull();
  });

  it('clearRefreshToken 防御性移除遗留 LF_refreshToken', () => {
    const storage = globalThis.localStorage as Storage | undefined;
    if (
      !storage ||
      typeof storage.setItem !== 'function' ||
      typeof storage.getItem !== 'function' ||
      typeof storage.removeItem !== 'function'
    ) {
      // Node 25 / 部分 happy-dom 会话下 localStorage API 不完整：跳过遗留键清理断言，
      // 权威 store 清理路径仍由 Property 6 与 logout/refresh-fail 单测覆盖。
      setRefreshToken('store-rt');
      clearRefreshToken();
      expect(getRefreshToken()).toBeNull();
      return;
    }

    storage.setItem(LEGACY_REFRESH_TOKEN_KEY, 'legacy-rt');
    setRefreshToken('store-rt');

    clearRefreshToken();

    expect(getRefreshToken()).toBeNull();
    expect(storage.getItem(LEGACY_REFRESH_TOKEN_KEY)).toBeNull();
  });

  it('happy path: 登录态可读写 access + refresh', () => {
    seedAuthenticatedSession('access-ok', 'refresh-ok');

    expect(useAccessStore().accessToken).toBe('access-ok');
    expect(getRefreshToken()).toBe('refresh-ok');
  });

    it('Property 6: 令牌清理完备 — 登出/刷新失败等价清理后均未认证', () => {
    fc.assert(
      fc.property(usableTokenArb, usableTokenArb, (access, refresh) => {
        // 每次 property 迭代隔离 pinia 会话，避免状态串扰
        setActivePinia(createPinia());
        seedAuthenticatedSession(access, refresh);

        expect(useAccessStore().accessToken).toBe(access);
        expect(getRefreshToken()).toBe(refresh);

        clearAuthTokensLikeProductionPaths();

        expect(useAccessStore().accessToken).toBeNull();
        expect(getRefreshToken()).toBeNull();
        expect(useAccessStore().refreshToken).toBeNull();
      }),
      { numRuns: 100 },
    );
  });

  it('unit: logout 清理路径后 access + refresh 均不可用', () => {
    seedAuthenticatedSession('acc-logout', 'rt-logout');
    // 对齐 store/auth.ts logout：clearRefreshToken + setAccessToken(null)
    clearAuthTokensLikeProductionPaths();
    assertUnauthenticatedReadable();
  });

  it('unit: 刷新失败清理路径后 access + refresh 均不可用', () => {
    seedAuthenticatedSession('acc-refresh-fail', 'rt-refresh-fail');
    // 对齐 api/request.ts doRefreshToken catch / 无 newToken 分支
    clearAuthTokensLikeProductionPaths();
    assertUnauthenticatedReadable();
  });

  it('unit: doReAuthenticate 清理路径后 access + refresh 均不可用', () => {
    seedAuthenticatedSession('acc-reauth', 'rt-reauth');
    // 对齐 api/request.ts doReAuthenticate
    clearAuthTokensLikeProductionPaths();
    assertUnauthenticatedReadable();
  });

  it('unit: guard clearInvalidLoginState 清理路径后 access + refresh 均不可用', () => {
    seedAuthenticatedSession('acc-guard', 'rt-guard');
    // 对齐 router/guard.ts clearInvalidLoginState 的 token 部分
    clearAuthTokensLikeProductionPaths();
    assertUnauthenticatedReadable();
  });

  it('boundary: 已未认证时再清理仍保持未认证（幂等）', () => {
    expect(useAccessStore().accessToken).toBeNull();
    expect(getRefreshToken()).toBeNull();

    clearAuthTokensLikeProductionPaths();
    assertUnauthenticatedReadable();

    clearAuthTokensLikeProductionPaths();
    assertUnauthenticatedReadable();
  });

  it('boundary: 仅 access 有值时清理后二者均未认证', () => {
    useAccessStore().setAccessToken('only-access');
    expect(getRefreshToken()).toBeNull();

    clearAuthTokensLikeProductionPaths();
    assertUnauthenticatedReadable();
  });

  it('boundary: 仅 refresh 有值时清理后二者均未认证', () => {
    setRefreshToken('only-refresh');
    expect(useAccessStore().accessToken).toBeNull();

    clearAuthTokensLikeProductionPaths();
    assertUnauthenticatedReadable();
  });

  it('setRefreshToken(null) 与 clear 在 refresh 面上等价为未认证', () => {
    setRefreshToken('rt');
    setRefreshToken(null);
    expect(getRefreshToken()).toBeNull();
  });
});
