import { Outlet } from '@umijs/max';

/**
 * 受保护布局壳占位。
 *
 * 当前仅渲染路由出口 <Outlet />，保证常量路由壳可加载（Req 1.3）。
 * 真正的 ProLayout（侧边栏/顶栏/多页签）与登录守卫在后续任务
 * （8.1 布局系统、7.4 路由守卫）中实现。
 */
export default function ShellLayout() {
  return <Outlet />;
}
