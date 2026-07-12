import fs from 'node:fs';
import path from 'node:path';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';

import Vue from '@vitejs/plugin-vue';
import VueJsx from '@vitejs/plugin-vue-jsx';
import type { Plugin } from 'vite';
import { configDefaults, defineConfig } from 'vitest/config';

const rootDir = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);

function resolveDayjsRoot(): string {
  try {
    return path.dirname(require.resolve('dayjs/package.json'));
  } catch {
    return path.resolve(
      rootDir,
      'node_modules/.pnpm/dayjs@1.11.21/node_modules/dayjs',
    );
  }
}

const dayjsRoot = resolveDayjsRoot();

/**
 * Rewrite bare `dayjs/plugin/*` and `dayjs/locale/*` imports to real files.
 * Required because @v-c/picker ships ESM that imports extension-less plugin paths,
 * which Node cannot resolve against dayjs@1.11.x packaging.
 */
function vitestDayjsBareImportPlugin(): Plugin {
  const rewrite = (source: string): string | null => {
    const pluginMatch = source.match(/^dayjs\/plugin\/([^/]+?)(?:\.js)?$/);
    if (pluginMatch) {
      return path.join(dayjsRoot, 'plugin', `${pluginMatch[1]}.js`);
    }
    const localeMatch = source.match(/^dayjs\/locale\/([^/]+?)(?:\.js)?$/);
    if (localeMatch) {
      return path.join(dayjsRoot, 'locale', `${localeMatch[1]}.js`);
    }
    if (source === 'dayjs') {
      // Prefer CJS build for node-side vitest stability
      const cjs = path.join(dayjsRoot, 'dayjs.min.js');
      if (fs.existsSync(cjs)) return cjs;
    }
    return null;
  };

  return {
    name: 'vitest-dayjs-bare-import',
    enforce: 'pre',
    resolveId(source) {
      return rewrite(source);
    },
  };
}

export default defineConfig({
  plugins: [vitestDayjsBareImportPlugin(), Vue(), VueJsx()],
  ssr: {
    // Ensure picker is transformed by Vite (not loaded as raw external ESM).
    noExternal: true,
  },
  test: {
    environment: 'happy-dom',
    environmentOptions: {
      happyDOM: {
        settings: {
          // happy-dom v20+ disables JS evaluation by default (security fix).
          // Treat disabled script loading as success to preserve test behavior.
          handleDisabledFileLoadingAsSuccess: true,
        },
      },
    },
    server: {
      deps: {
        // Inline all deps so resolve hooks apply to @v-c/picker → dayjs chains.
        inline: true,
      },
    },
    exclude: [
      ...configDefaults.exclude,
      '**/e2e/**',
      '**/dist/**',
      '**/.{idea,git,cache,output,temp}/**',
      '**/node_modules/**',
      '**/{stylelint,eslint}.config.*',
      '**/{oxfmt,oxlint}.config.*',
    ],
  },
});
