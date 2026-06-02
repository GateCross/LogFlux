/**
 * 常量路由壳（构建期静态存在）。
 *
 * 设计依据（design.md「路由策略」）：Umi 配置式路由为构建期静态，
 * 无法直接表达「登录后按角色注入」。因此此处仅声明：
 *  - 内置页：login / 403 / 404 / 500 / iframe（满足 Req 7、Req 4.4/4.8 的重定向落点）
 *  - 一个受保护的 `layout` 壳路由：登录后由 app.tsx 的 patchClientRoutes
 *    依据 /api/route/getUserRoutes 结果，将动态路由注入其 children（见任务 7.4）。
 *
 * 注意：`wrappers` / 守卫 与动态注入逻辑在后续任务（7.x / 8.x）补充，
 * 本任务仅落地常量路由壳与页面占位，保证工程可构建。
 */
export default [
  // 根路径重定向到首页
  { path: '/', redirect: '/dashboard' },

  // ---- 内置页（不需要登录布局） ----
  { name: 'login', path: '/login', layout: false, component: '@/pages/_builtin/login' },
  { name: '403', path: '/403', layout: false, component: '@/pages/_builtin/403' },
  { name: '404', path: '/404', layout: false, component: '@/pages/_builtin/404' },
  { name: '500', path: '/500', layout: false, component: '@/pages/_builtin/500' },
  { name: 'iframe', path: '/iframe-page/:url', layout: false, component: '@/pages/_builtin/iframe-page' },

  // ---- 受保护布局壳：动态路由运行时注入到此 children ----
  {
    path: '/',
    component: '@/layouts/index',
    routes: [
      { name: 'dashboard', path: '/dashboard', component: '@/pages/dashboard' },
      // Catch-all: 动态路由兜底，由 DynamicPage 组件根据路由模型解析页面
      { path: '/*', component: '@/pages/_builtin/dynamic-page' },
    ],
  },

  // 兜底 404
  { path: '/*', component: '@/pages/_builtin/404', layout: false },
];
