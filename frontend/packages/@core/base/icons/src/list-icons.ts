/**
 * offline 版 @iconify/vue 不提供 listIcons。
 * 这里维护已注册图标名称，供 IconPicker 等本地枚举使用。
 */

const iconsByPrefix = new Map<string, string[]>();

/** 记录某个前缀下的图标名（形如 prefix:name） */
export function registerIconNames(prefix: string, names: string[]) {
  const fullNames = names.map((name) =>
    name.includes(':') ? name : `${prefix}:${name}`,
  );
  iconsByPrefix.set(prefix, fullNames);
}

/**
 * 兼容原 @iconify/vue listIcons(provider, prefix) 调用约定。
 * icon-picker 中调用：listIcons('', props.prefix)
 */
export function listIcons(_provider = '', prefix = ''): string[] {
  if (prefix) {
    return iconsByPrefix.get(prefix) ?? [];
  }
  return [...iconsByPrefix.values()].flat();
}
