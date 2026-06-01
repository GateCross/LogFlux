import { resolve } from 'node:path';
import { defineConfig } from 'vitest/config';

/**
 * Vitest 测试配置（任务 1.3）。
 *
 * - 运行器：vitest；环境：jsdom（承载 @testing-library/react 组件测试）。
 * - 路径别名：与 UmiJS 应用保持一致（`@/*` → `src/*`、`@@/*` → `src/.umi/*`），
 *   确保被测纯逻辑/组件在测试中按与运行时相同的方式解析模块。
 * - JSX：交由 vitest 内置 esbuild 以自动运行时（react-jsx）转译，无需 babel/react-refresh，
 *   避免测试环境中的 react-refresh 预置问题。
 * - 属性测试默认 `{ numRuns: 100 }` 见 `src/test/pbt.ts`。
 */
export default defineConfig({
  resolve: {
    alias: [
      { find: /^@\/(.*)$/, replacement: `${resolve(__dirname, 'src')}/$1` },
      { find: /^@@\/(.*)$/, replacement: `${resolve(__dirname, 'src/.umi')}/$1` },
    ],
  },
  esbuild: {
    jsx: 'automatic',
    jsxImportSource: 'react',
  },
  test: {
    environment: 'jsdom',
    globals: false,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    exclude: ['node_modules', 'dist', 'src/.umi', 'src/.umi-production', 'src/.umi-test'],
    css: false,
  },
});
