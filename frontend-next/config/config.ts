import { defineConfig } from '@umijs/max';
import proxy from './proxy';
import routes from './routes';

/**
 * 子路径部署支持（Req 16.5 / 16.8）：
 * - `base`：前端路由前缀（等价旧 VITE_BASE_URL）
 * - `publicPath`：静态资源加载前缀
 * 取自构建期变量 `UMI_APP_BASE_URL`，未配置时默认根路径 `/`。
 */
const BASE_URL = process.env.UMI_APP_BASE_URL || '/';

export default defineConfig({
  /** 路由前缀，未配置默认 `/`（Req 16.8） */
  base: BASE_URL,
  /** 资源前缀，与 base 一致以支持子路径部署下的资源解析（Req 16.5） */
  publicPath: BASE_URL,

  /** 常量路由壳：内置页 + 受保护布局壳（动态路由运行时注入，见 app.tsx / 任务 7.4） */
  routes,

  /** React 18 并发特性 */
  npmClient: 'pnpm',

  /** 内建插件：Ant Design 5 组件库 */
  antd: {},

  /**
   * 不启用 Umi 内置 ProLayout 插件。
   * 项目使用自定义布局壳 `src/layouts/index.tsx`（ProLayout + 多标签 + 主题切换 + 语言切换），
   * 开启内置插件会导致双重 ProLayout 嵌套，侧边栏和顶栏重复渲染。
   */
  layout: false,

  /** 内建插件：RBAC 权限（access.ts 定义权限点） */
  access: {},

  /** 内建插件：国际化（仅 zh-CN / en-US，默认 zh-CN，缺键回退 zh-CN） */
  locale: {
    default: 'zh-CN',
    baseSeparator: '-',
    antd: true,
    title: false,
    useLocalStorage: true,
  },

  /** 内建插件：统一请求层（后续在 app.tsx 注入拦截器复刻请求行为） */
  request: {},

  /** 开发期代理：将 `/api` 转发到后端（生产期由 Caddy 反代），仅 dev 生效 */
  proxy,

  /** 内建插件：全局初始状态（承载用户信息 / 偏好 / 路由初始化状态） */
  initialState: {},

  /** 数据流插件（src/models/* 全局状态） */
  model: {},

  /** TypeScript + React 运行时 */
  hash: true,
  history: { type: 'browser' },
});
