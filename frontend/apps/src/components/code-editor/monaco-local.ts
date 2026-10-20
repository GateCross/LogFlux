/**
 * Monaco 编辑器本地化配置。
 * 只注册编辑器核心与 shell 语法，代码与 web worker 全部打进构建产物，
 * 运行时不请求任何外网 CDN。
 */
import { loader } from '@guolao/vue-monaco-editor';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api.js';
import 'monaco-editor/esm/vs/editor/editor.all.js';
import 'monaco-editor/esm/vs/basic-languages/shell/shell.contribution.js';
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';

// 缺省时 monaco 会去 CDN 拉 workerMain.js，这里固定为打包后的本地 worker
globalThis.MonacoEnvironment = {
  getWorker: () => new EditorWorker(),
};

// 直接把实例交给 loader，跳过它注入 <script> 加载 vs 目录的流程
loader.config({ monaco });

export { monaco };
