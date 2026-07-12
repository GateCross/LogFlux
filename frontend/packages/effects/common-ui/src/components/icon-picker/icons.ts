import type { Recordable } from '@vben/types';

import { listIcons } from '@vben/icons';

/**
 * 一个缓存对象，在不刷新页面时，无需重复请求
 */
export const ICONS_MAP: Recordable<string[]> = {};

/**
 * 从本地已注册的离线图标集获取图标列表。
 * 不再请求 https://api.iconify.design。
 * @param prefix 图标集名称
 * @returns 图标集中包含的所有图标名称（prefix:name）
 */
export async function fetchIconsData(prefix: string): Promise<string[]> {
  if (Reflect.has(ICONS_MAP, prefix) && ICONS_MAP[prefix]) {
    return ICONS_MAP[prefix];
  }

  const icons = listIcons('', prefix);
  ICONS_MAP[prefix] = icons;
  if (icons.length === 0) {
    console.warn(
      `[icon-picker] 本地未注册图标集: ${prefix}，请在 packages/icons 中 addCollection`,
    );
  }
  return icons;
}
