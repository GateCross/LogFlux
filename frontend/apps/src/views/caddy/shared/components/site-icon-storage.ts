import { getSiteIconApi } from '#/api/caddy/server';

const STORAGE_PREFIX = 'logflux:site-icon:';

function getStorage(): Storage | null {
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      return window.localStorage;
    }
    if (typeof localStorage !== 'undefined') {
      return localStorage;
    }
  } catch {
    // 跨域或安全受限环境容错
  }
  return null;
}

/**
 * 从本地存储读取已缓存的站点图标 data URL。
 * 避免目录页加载时对所有站点发起大量回源探测。
 */
export function getStoredSiteIcon(key: string): string {
  const normalized = key.trim().toLowerCase();
  if (!normalized) return '';
  const storage = getStorage();
  if (!storage) return '';
  try {
    return storage.getItem(`${STORAGE_PREFIX}${normalized}`) || '';
  } catch {
    return '';
  }
}

/** 写入本地站点图标缓存 */
export function setStoredSiteIcon(key: string, icon: string): void {
  const normalized = key.trim().toLowerCase();
  if (!normalized) return;
  const storage = getStorage();
  if (!storage) return;
  try {
    if (icon) {
      storage.setItem(`${STORAGE_PREFIX}${normalized}`, icon);
    } else {
      storage.removeItem(`${STORAGE_PREFIX}${normalized}`);
    }
  } catch {
    // 忽略存储超额异常
  }
}

/**
 * 抓取并持久化站点图标：优先直连上游地址，没有上游才回落到域名。
 * 两个键都写一份，卡片改过上游后仍能命中旧图。
 */
export async function fetchAndStoreSiteIcon(
  upstream: string,
  host: string,
): Promise<string> {
  const target = upstream.trim() || host.trim();
  if (!target) return '';
  try {
    const icon = await getSiteIconApi({ upstream: upstream.trim(), host: host.trim() });
    if (icon) {
      setStoredSiteIcon(upstream, icon);
      setStoredSiteIcon(host, icon);
    }
    return icon || '';
  } catch {
    return '';
  }
}
