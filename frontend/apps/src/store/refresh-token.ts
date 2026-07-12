import { useAccessStore } from '@vben/stores';

const LEGACY_REFRESH_TOKEN_KEY = 'LF_refreshToken';

/**
 * 一次性清理历史双写 key；完成态只认 accessStore.refreshToken
 */
function purgeLegacyRefreshTokenKey() {
  try {
    localStorage.removeItem(LEGACY_REFRESH_TOKEN_KEY);
  } catch {
    // localStorage 不可用时忽略
  }
}

// 模块加载时执行一次遗留 key 清理（非兼容读窗口）
purgeLegacyRefreshTokenKey();

/** 读写 refreshToken（accessStore） */
export function getRefreshToken(): null | string {
  return useAccessStore().refreshToken;
}

export function setRefreshToken(token: null | string): void {
  useAccessStore().setRefreshToken(token);
}

export function clearRefreshToken(): void {
  useAccessStore().setRefreshToken(null);
  // 防御性清理：确保历史 key 不再残留
  purgeLegacyRefreshTokenKey();
}
