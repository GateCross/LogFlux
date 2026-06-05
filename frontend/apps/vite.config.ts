import { fileURLToPath } from 'node:url';
import { defineConfig } from '@vben/vite-config';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      resolve: {
        preserveSymlinks: false,
        alias: {
          'crypto-js': `${__dirname}node_modules/crypto-js`,
        },
      },
      server: {
        proxy: {
          '/api': {
            changeOrigin: true,
            target: 'http://127.0.0.1:8888',
            ws: true,
          },
        },
      },
    },
  };
});
