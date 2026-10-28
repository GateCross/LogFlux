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
export function getStoredSiteIcon(host: string): string {
  const normalized = host.trim().toLowerCase();
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
export function setStoredSiteIcon(host: string, icon: string): void {
  const normalized = host.trim().toLowerCase();
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
 * 抓取并持久化站点图标。
 * 首次建站或用户手动点击图标时调用，抓到后写入 localStorage。
 */
export async function fetchAndStoreSiteIcon(host: string): Promise<string> {
  const normalized = host.trim().toLowerCase();
  if (!normalized) return '';
  try {
    const icon = await getSiteIconApi(normalized);
    if (icon) {
      setStoredSiteIcon(normalized, icon);
    }
    return icon || '';
  } catch {
    return '';
  }
}
