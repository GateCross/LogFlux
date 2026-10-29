import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  fetchAndStoreSiteIcon,
  getStoredSiteIcon,
  setStoredSiteIcon,
} from './site-icon-storage';

vi.mock('#/api/caddy/server', () => ({
  getSiteIconApi: vi.fn(async (host: string) => {
    if (host === 'example.com') {
      return 'data:image/png;base64,mockIconData';
    }
    return '';
  }),
}));

describe('site-icon-storage', () => {
  beforeEach(() => {
    const store = new Map<string, string>();
    const mockStorage: Storage = {
      clear: () => store.clear(),
      getItem: (key: string) => store.get(key) ?? null,
      key: (index: number) => Array.from(store.keys())[index] ?? null,
      length: 0,
      removeItem: (key: string) => store.delete(key),
      setItem: (key: string, value: string) => store.set(key, String(value)),
    };
    Object.defineProperty(globalThis, 'localStorage', {
      value: mockStorage,
      writable: true,
      configurable: true,
    });
    globalThis.localStorage.clear();
  });

  it('读取与写入本地图标缓存', () => {
    expect(getStoredSiteIcon('example.com')).toBe('');
    setStoredSiteIcon('example.com', 'data:image/png;base64,abc');
    expect(getStoredSiteIcon('example.com')).toBe('data:image/png;base64,abc');
    // 大小写与空格不敏感
    expect(getStoredSiteIcon('  EXAMPLE.COM ')).toBe('data:image/png;base64,abc');

    // 写入空值视为删除
    setStoredSiteIcon('example.com', '');
    expect(getStoredSiteIcon('example.com')).toBe('');
  });

  it('fetchAndStoreSiteIcon 抓到后写入本地存储', async () => {
    const icon = await fetchAndStoreSiteIcon('example.com');
    expect(icon).toBe('data:image/png;base64,mockIconData');
    expect(getStoredSiteIcon('example.com')).toBe('data:image/png;base64,mockIconData');

    const empty = await fetchAndStoreSiteIcon('notfound.com');
    expect(empty).toBe('');
    expect(getStoredSiteIcon('notfound.com')).toBe('');
  });
});
