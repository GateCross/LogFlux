/**
 * Monaco 编辑器本地化配置。
 * 编辑器代码与 shell 语法随构建产物打包，worker 走本地 URL，
 * 运行时不请求任何外网 CDN。
 */
import 'monaco-editor/esm/vs/editor/editor.all.js';
import 'monaco-editor/esm/vs/basic-languages/shell/shell.contribution.js';

// 缺省时 monaco 会去 CDN 拉 workerMain.js，这里固定为打包后的本地 worker
globalThis.MonacoEnvironment = {
  getWorker: () =>
    new Worker(
      new URL('monaco-editor/esm/vs/editor/editor.worker.js', import.meta.url),
      { type: 'module' },
    ),
};
