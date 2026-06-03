import { requestClient } from '#/api/request';

export namespace RouteApi {
  /** 后端返回的路由节点 */
  export interface MenuRoute {
    children?: MenuRoute[];
    component: string;
    icon?: string;
    id: number;
    meta?: Record<string, any>;
    order?: number;
    parentId: number;
    path: string;
    title: string;
  }

  /** 用户路由响应 */
  export interface UserRoute {
    home: string;
    routes: MenuRoute[];
  }
}

/**
 * 获取常量路由 — GET /route/getConstantRoutes
 */
export async function getConstantRoutesApi() {
  return requestClient.get<RouteApi.MenuRoute[]>(
    '/route/getConstantRoutes',
  );
}

/**
 * 获取用户路由 — GET /route/getUserRoutes
 */
export async function getUserRoutesApi() {
  return requestClient.get<RouteApi.UserRoute>('/route/getUserRoutes');
}

/**
 * 检查路由是否存在 — GET /route/isRouteExist
 */
export async function isRouteExistApi(routeName: string) {
  return requestClient.get<boolean>(
    `/route/isRouteExist?routeName=${routeName}`,
  );
}
