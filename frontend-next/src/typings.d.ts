/**
 * 全局类型声明（迁移自旧 Vue 版 `src/typings/api/*` 与 `src/typings/app.d.ts`）。
 *
 * 迁移原则（task 1.2 / design.md「Data Models」）：
 *  - 保留 `Api.*` / `App.*` 命名空间与统一响应结构。
 *  - 剥离旧版对 Vue / Naive UI / `@elegant-router` / `@sa/color` 的类型耦合，
 *    使类型在 React + Umi Max 环境下自洽、可独立编译。
 *  - 路由相关类型按后端契约（`backend/api/route.api`）建模，保证与真实返回结构一致。
 */

/**
 * 后端统一响应结构（design.md「Data Models」）。
 *
 * 与后端 go-zero `{ code, msg, message?, data? }` 对齐：
 *  - `message` 为新字段，`msg` 为兼容旧字段（取值时 `message || msg`）。
 */
type BackendResponse<T = unknown> = {
  /** 业务响应码 */
  code: string | number;
  /** 响应消息（新字段） */
  message?: string;
  /** 响应消息（兼容旧字段） */
  msg?: string;
  /** 响应数据 */
  data: T;
};

/**
 * Namespace Api
 *
 * 全部后端接口相关类型（迁移自 `src/typings/api/*`）。
 */
declare namespace Api {
  /**
   * namespace Common
   *
   * 通用分页与记录类型。
   */
  namespace Common {
    /** 分页通用参数 */
    interface PaginatingCommonParams {
      /** 当前页码 */
      current: number;
      /** 每页条数 */
      size: number;
      /** 总条数 */
      total: number;
    }

    /** 分页查询结果通用结构 */
    interface PaginatingQueryRecord<T = any> extends PaginatingCommonParams {
      records: T[];
    }

    /** 表格通用搜索参数 */
    type CommonSearchParams = Pick<Common.PaginatingCommonParams, 'current' | 'size'>;

    /**
     * 启用状态
     *
     * - "1": 启用
     * - "2": 禁用
     */
    type EnableStatus = '1' | '2';

    /** 通用记录（含审计字段） */
    type CommonRecord<T = any> = {
      /** 记录 id */
      id: number;
      /** 创建人 */
      createBy: string;
      /** 创建时间 */
      createTime: string;
      /** 更新人 */
      updateBy: string;
      /** 更新时间 */
      updateTime: string;
      /** 记录状态 */
      status: EnableStatus | null;
    } & T;
  }

  /**
   * namespace Auth
   *
   * backend api module: "auth"
   */
  namespace Auth {
    interface LoginToken {
      token: string;
      refreshToken: string;
    }

    interface UserInfo {
      userId: string | number;
      username: string;
      roles: string[];
      buttons?: string[];
      /** 用户偏好 JSON 字符串，对应 Preferences_Store（Req 14 / 15） */
      preferences?: string;
    }
  }

  /**
   * namespace Role
   *
   * backend api module: "role"
   */
  namespace Role {
    interface RoleItem {
      id: number;
      /** 唯一标识，如 "admin" */
      name: string;
      /** 显示名称，如 "管理员" */
      displayName: string;
      description: string;
      /** 权限列表 */
      permissions: string[];
      createdAt: string;
    }

    interface RoleListResp {
      list: RoleItem[];
    }

    interface UpdateRolePermissionsReq {
      id: number;
      permissions: string[];
    }
  }

  /**
   * namespace Route
   *
   * backend api module: "route"。按 `backend/api/route.api` 契约建模，
   * 取代旧版对 `@elegant-router/types` 的依赖。
   */
  namespace Route {
    /** 路由元数据（对应后端 RouteMeta） */
    interface RouteMeta {
      /** 菜单标题 */
      title: string;
      /** i18n 文案键 */
      i18nKey?: string;
      /** 图标（iconify） */
      icon?: string;
      /** 本地图标 */
      localIcon?: string;
      /** 排序 */
      order?: number;
      /** 是否在菜单中隐藏 */
      hideInMenu?: boolean;
      /** 可访问该路由的角色集合 */
      roles?: string[];
      /** 外链地址（配置后在新标签打开） */
      href?: string;
      /** 其他扩展元数据 */
      [key: string]: unknown;
    }

    /** 菜单路由（对应后端 MenuRoute，可递归嵌套子路由） */
    interface MenuRoute {
      /** 路由名 */
      name: string;
      /** 路由路径 */
      path: string;
      /** 组件标识（如 `view.security_policy`），由运行时解析为页面 import */
      component?: string;
      /** 路由元数据 */
      meta: RouteMeta;
      /** 子路由 */
      children?: MenuRoute[];
    }

    /** 用户动态路由响应（对应后端 UserRouteResp） */
    interface UserRoute {
      /** 首页路由名 */
      home: string;
      /** 路由树 */
      routes: MenuRoute[];
    }
  }
}

/**
 * Namespace App
 *
 * 应用级类型（迁移自 `src/typings/app.d.ts`，保留框架无关部分）。
 */
declare namespace App {
  /** Service 命名空间 */
  namespace Service {
    /**
     * 后端服务响应数据（统一结构）。
     *
     * 与全局 `BackendResponse<T>` 等价，保留 `App.Service.Response` 命名以兼容旧调用方。
     */
    type Response<T = unknown> = BackendResponse<T>;
  }
}
