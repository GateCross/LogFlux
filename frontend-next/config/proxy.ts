/**
 * 开发期代理配置（等价旧 Vue 版 `VITE_HTTP_PROXY`）。
 *
 * 设计依据：design.md「Migration Strategy / 并存与替换」——
 * 开发期通过 Umi `proxy` 将 `/api` 转发到后端 go-zero 服务，生产期改由 Caddy `reverse_proxy /api/*` 承担。
 *
 * 代理目标取自构建期变量 `UMI_APP_PROXY_TARGET`，未配置时默认后端默认监听地址 `http://127.0.0.1:8888`。
 */
const PROXY_TARGET = process.env.UMI_APP_PROXY_TARGET || 'http://127.0.0.1:8888';

export default {
  '/api': {
    target: PROXY_TARGET,
    changeOrigin: true,
    // 后端路由本身带 `/api` 前缀，保持路径不变（不做 rewrite）。
  },
};
