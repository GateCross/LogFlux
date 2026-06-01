/**
 * 应用级运行时常量（可经环境变量覆盖）。
 *
 * 设计依据：design.md「环境变量映射」。
 */

/** 超级角色标识（Req 4.5）：拥有该角色的用户放行所有 RBAC 鉴权。等价旧 VITE_STATIC_SUPER_ROLE。 */
export const SUPER_ROLE = process.env.UMI_APP_STATIC_SUPER_ROLE ?? 'R_SUPER';

/** 应用首页路由名（Req 4.8）。等价旧 VITE_ROUTE_HOME。 */
export const ROUTE_HOME = process.env.UMI_APP_ROUTE_HOME ?? 'dashboard';

/** 动态路由模式。等价旧 VITE_AUTH_ROUTE_MODE。 */
export const AUTH_ROUTE_MODE = process.env.UMI_APP_AUTH_ROUTE_MODE ?? 'dynamic';

/** 本地存储 key 前缀（Req 15.1）：用于隔离不同站点。等价旧 VITE_STORAGE_PREFIX。 */
export const STORAGE_PREFIX = process.env.UMI_APP_STORAGE_PREFIX ?? 'LF_';

/**
 * 后端服务基础地址（Req 2.1 / 16.3 / 16.7）。等价旧 VITE_SERVICE_BASE_URL。
 * 留空表示采用同源相对路径（请求落到 `/api/*`，由代理 / Caddy 反代到后端）。
 */
export const SERVICE_BASE_URL = process.env.UMI_APP_SERVICE_BASE_URL ?? '';
