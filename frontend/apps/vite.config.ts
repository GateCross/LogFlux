import { fileURLToPath } from 'node:url';

import { defineConfig } from '@vben/vite-config';

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      resolve: {
        alias: {
          // 原 loader 内置 jsdelivr CDN 地址，替换为直接返回本地实例的实现
          '@monaco-editor/loader': fileURLToPath(
            new URL(
              './src/components/code-editor/monaco-loader.ts',
              import.meta.url,
            ),
          ),
        },
        preserveSymlinks: false,
      },
      server: {
        proxy: {
          '/api': {
            changeOrigin: true,
            // Use localhost (not 127.0.0.1): on this machine 127.0.0.1:8888 is
            // claimed by kiro-proxy, while logflux listens on 0.0.0.0 / [::]:8888.
            // Hitting 127.0.0.1 causes Vite "socket hang up" on /api/*.
            target: 'http://localhost:8888',
            ws: true,
          },
        },
      },
    },
  };
});
