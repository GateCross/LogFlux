/**
 * 路由 Service_API（迁移自旧 Vue 版 `frontend/src/service/api/route.ts`）。
 *
 * 迁移说明：
 *  - 端点与旧版逐一对齐：`GET /api/route/getConstantRoutes`、
 *    `GET /api/route/getUserRoutes`、`GET /api/route/isRouteExist`，与
 *    `backend/api/route.api` 契约一致。
 *  - 全部调用统一经 Request_Layer（`src/utils/request`），返回扁平结构 `{ data, error }`；
 *    Request_Layer 已配置 10s 默认超时、鉴权头注入与响应码处理，
 *    本层只负责声明端点与请求参数，不重复处理副作用。
 */
import { request } from '@/utils/request';
import type { FlatResponse } from '@/utils/request';

/**
 * 获取常量路由（公开页面，无需鉴权）（`GET /api/route/getConstantRoutes`）。
 *
 * @returns 常量菜单路由集合
 */
export function fetchGetConstantRoutes(): Promise<FlatResponse<Api.Route.MenuRoute[]>> {
  return request<Api.Route.MenuRoute[]>({ url: '/api/route/getConstantRoutes' });
}

/**
 * 获取当前用户动态路由（按角色权限下发）（`GET /api/route/getUserRoutes`）。
 *
 * @returns 用户路由响应（含首页路由名与路由树）
 */
export function fetchGetUserRoutes(): Promise<FlatResponse<Api.Route.UserRoute>> {
  return request<Api.Route.UserRoute>({ url: '/api/route/getUserRoutes' });
}

/**
 * 查询指定路由是否存在（`GET /api/route/isRouteExist`）。
 *
 * @param routeName 路由名称
 * @returns 路由是否存在
 */
export function fetchIsRouteExist(routeName: string): Promise<FlatResponse<boolean>> {
  return request<boolean>({
    url: '/api/route/isRouteExist',
    params: { routeName },
  });
}
