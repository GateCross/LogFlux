import { defineComponent, h } from 'vue';

// 使用 offline 入口，避免运行时请求 Iconify CDN
import { Icon } from '@iconify/vue/offline';

function createIconifyIcon(icon: string) {
  return defineComponent({
    name: `Icon-${icon}`,
    setup(props, { attrs }) {
      return () => h(Icon, { icon, ...props, ...attrs });
    },
  });
}

export { createIconifyIcon };
