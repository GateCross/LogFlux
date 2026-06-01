/**
 * RBAC 权限定义（Umi access 插件）。
 *
 * 占位实现：完整的角色/超级角色判定（canAccess、R_SUPER 放行）见任务 7.x。
 */
export default function access(initialState: { currentUser?: { roles?: string[] } } | undefined) {
  const roles = initialState?.currentUser?.roles ?? [];
  return {
    isSuperRole: roles.includes('R_SUPER'),
  };
}
