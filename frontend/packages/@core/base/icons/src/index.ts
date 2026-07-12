export * from './create-icon';

export * from './lucide';

// 使用 offline 入口：仅渲染本地已注册图标，不会请求外部 API
export type { IconifyIcon as IconifyIconStructure } from '@iconify/vue/offline';
export {
  addCollection,
  addIcon,
  Icon as IconifyIcon,
} from '@iconify/vue/offline';

export { listIcons, registerIconNames } from './list-icons';
