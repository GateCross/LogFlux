/** list-detail 请求/错误辅助：抑制全局 toast，页内展示错误 */
import type { RequestClientConfig } from '@vben/request';

import { apiErrorMessage } from '#/utils/api-error-message';

import { suppressGlobalErrorToast } from './request';

export { suppressGlobalErrorToast };

/** 合并请求配置，强制 errorMessageMode: 'none' */
export function withListDetailErrorMode<
  T extends RequestClientConfig | Record<string, unknown> = RequestClientConfig,
>(config?: T): T & typeof suppressGlobalErrorToast {
  return {
    ...(config as T),
    ...suppressGlobalErrorToast,
  };
}

/** 页内错误文案（Alert/表格），勿再 message.error */
export function listDetailErrorMessage(
  error: unknown,
  fallback: string,
): string {
  return apiErrorMessage(error, fallback);
}

/** 从 query.error 派生页内错误；无 error 时返回 null */
export function toListDetailErrorMessage(
  error: unknown | null | undefined,
  fallback: string,
): string | null {
  if (error == null) return null;
  return apiErrorMessage(error, fallback);
}

/** 是否为空列表数据 */
export function isListDetailEmpty(data: unknown): boolean {
  if (data == null) return true;
  if (Array.isArray(data)) return data.length === 0;
  if (typeof data === 'object' && data !== null && 'list' in data) {
    const list = (data as { list?: unknown }).list;
    return !Array.isArray(list) || list.length === 0;
  }
  return false;
}
