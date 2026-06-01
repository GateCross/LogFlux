/**
 * 全局测试环境初始化（vitest setupFiles，任务 1.3）。
 *
 * - 引入 `@testing-library/jest-dom` 的自定义匹配器（如 `toBeInTheDocument`）。
 * - 每个测试用例后自动卸载已渲染组件，避免跨用例的 DOM 残留。
 * - 修复 Web Storage：部分 Node 版本会暴露一个实验性的全局 `localStorage`，它会遮蔽 jsdom 的
 *   实现且不具备 `setItem/getItem/clear` 等方法，导致依赖本地存储的测试报
 *   `localStorage.xxx is not a function`。此处在检测到不可用实现时，注入一个最小化的
 *   内存版 `Storage`，保证 Preferences_Store 等依赖本地存储的纯逻辑测试可运行。
 */
import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

/** 最小化内存版 Storage 实现，满足测试中对 localStorage 的使用。 */
function createMemoryStorage(): Storage {
  const store = new Map<string, string>();
  return {
    get length() {
      return store.size;
    },
    clear() {
      store.clear();
    },
    getItem(key: string) {
      return store.has(key) ? (store.get(key) as string) : null;
    },
    key(index: number) {
      return Array.from(store.keys())[index] ?? null;
    },
    removeItem(key: string) {
      store.delete(key);
    },
    setItem(key: string, value: string) {
      store.set(key, String(value));
    },
  } as Storage;
}

/** 当前 localStorage 实现是否可用（具备核心方法）。 */
function isStorageUsable(candidate: unknown): candidate is Storage {
  return (
    !!candidate &&
    typeof (candidate as Storage).setItem === 'function' &&
    typeof (candidate as Storage).getItem === 'function' &&
    typeof (candidate as Storage).clear === 'function'
  );
}

if (typeof window !== 'undefined' && !isStorageUsable((window as Window & typeof globalThis).localStorage)) {
  const memory = createMemoryStorage();
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: memory,
  });
}

afterEach(() => {
  cleanup();
});
