/**
 * 本地 monaco loader。
 * 通过 vite alias 替换 @monaco-editor/loader：原包内置 jsdelivr CDN 默认地址，
 * 并用注入 <script> 的方式异步加载 vs 目录。这里直接返回随包构建的实例。
 */
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api.js';

type Monaco = typeof monaco;

const localLoader = {
  /** 保留原接口签名，实例已本地化，无需额外配置 */
  config(_params: unknown): void {},
  init(): Promise<Monaco> {
    return Promise.resolve(monaco);
  },
  __getMonacoInstance(): Monaco {
    return monaco;
  },
};

export default localLoader;
