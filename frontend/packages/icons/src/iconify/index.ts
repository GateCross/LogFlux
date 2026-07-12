import { createIconifyIcon } from '@vben-core/icons';

import { setupIconifyOffline } from './load';

// 应用侧 import '@vben/icons' 时立即完成离线注册
setupIconifyOffline();

export * from '@vben-core/icons';
export { setupIconifyOffline } from './load';

export const MdiKeyboardEsc = createIconifyIcon('mdi:keyboard-esc');
