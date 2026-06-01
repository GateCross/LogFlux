/**
 * 全局测试环境初始化（vitest setupFiles，任务 1.3）。
 *
 * - 引入 `@testing-library/jest-dom` 的自定义匹配器（如 `toBeInTheDocument`）。
 * - 每个测试用例后自动卸载已渲染组件，避免跨用例的 DOM 残留。
 */
import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

afterEach(() => {
  cleanup();
});
