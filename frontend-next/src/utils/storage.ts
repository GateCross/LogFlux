/**
 * 本地存储封装（任务 3.4 / Req 6.5）。
 *
 * 统一带有 LF_ 前缀的 localStorage 操作，避免与同一域名下其他应用冲突。
 * 支持任意可序列化的 JSON 数据。
 */
import { STORAGE_PREFIX } from '@/constants/app';

/**
 * 获取带前缀的真实 key。
 */
function getKey(key: string): string {
  return `${STORAGE_PREFIX}${key}`;
}

/**
 * 从 localStorage 获取 JSON 数据。
 */
export function getStorage<T>(key: string): T | null {
  try {
    if (typeof localStorage === 'undefined') return null;
    const raw = localStorage.getItem(getKey(key));
    if (raw === null) return null;
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

/**
 * 保存 JSON 数据到 localStorage。
 */
export function setStorage<T>(key: string, value: T): void {
  try {
    if (typeof localStorage === 'undefined') return;
    if (value === null || value === undefined) {
      localStorage.removeItem(getKey(key));
    } else {
      localStorage.setItem(getKey(key), JSON.stringify(value));
    }
  } catch {
    /* 忽略跨域/无痕模式等导致抛错 */
  }
}

/**
 * 移除指定 key 的 localStorage 数据。
 */
export function removeStorage(key: string): void {
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem(getKey(key));
    }
  } catch {
    // ignore
  }
}

/**
 * 清空带有指定前缀的所有 localStorage 数据。
 */
export function clearStorage(): void {
  try {
    if (typeof localStorage !== 'undefined') {
      const keysToRemove: string[] = [];
      for (let i = 0; i < localStorage.length; i++) {
        const k = localStorage.key(i);
        if (k && k.startsWith(STORAGE_PREFIX)) {
          keysToRemove.push(k);
        }
      }
      keysToRemove.forEach(k => localStorage.removeItem(k));
    }
  } catch {
    // ignore
  }
}
