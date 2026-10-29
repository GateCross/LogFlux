import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  fetchAndStoreSiteIcon,
  getStoredSiteIcon,
  setStoredSiteIcon,
} from './site-icon-storage';

vi.mock('#/api/caddy/server', () => ({
  getSiteIconApi: vi.fn(async (params: { host?: string; upstream?: string }) =>
    params.upstream === '192.168.50.100:8006' || params.host === 'example.com'
      ? 'data:image/png;base64,mockIconData'
      : '',
  ),
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
    expect(getStoredSiteIcon('192.168.50.100:8006')).toBe('');
    setStoredSiteIcon('192.168.50.100:8006', 'data:image/png;base64,abc');
    expect(getStoredSiteIcon('192.168.50.100:8006')).toBe('data:image/png;base64,abc');
    // 大小写与空格不敏感
    expect(getStoredSiteIcon('  192.168.50.100:8006 ')).toBe('data:image/png;base64,abc');

    // 写入空值视为删除
    setStoredSiteIcon('192.168.50.100:8006', '');
    expect(getStoredSiteIcon('192.168.50.100:8006')).toBe('');
  });

  it('fetchAndStoreSiteIcon 直连上游抓取，上游与域名都写入缓存', async () => {
    const icon = await fetchAndStoreSiteIcon('192.168.50.100:8006', 'example.com');
    expect(icon).toBe('data:image/png;base64,mockIconData');
    expect(getStoredSiteIcon('192.168.50.100:8006')).toBe('data:image/png;base64,mockIconData');
    expect(getStoredSiteIcon('example.com')).toBe('data:image/png;base64,mockIconData');

    // 只有域名时按域名兜底，抓不到返回空串
    expect(await fetchAndStoreSiteIcon('', 'example.com')).toBe(
      'data:image/png;base64,mockIconData',
    );
    expect(await fetchAndStoreSiteIcon('', 'notfound.com')).toBe('');
    expect(getStoredSiteIcon('notfound.com')).toBe('');
  });
});
