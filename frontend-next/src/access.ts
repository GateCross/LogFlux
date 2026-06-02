/**
 * RBAC 权限定义（Umi access 插件）。
 *
 * 占位实现：完整的角色/超级角色判定（canAccess、SUPER_ROLE 放行）见任务 7.x。
 */
import { SUPER_ROLE } from '@/constants/app';

export default function access(initialState: { currentUser?: { roles?: string[] } } | undefined) {
  const roles = initialState?.currentUser?.roles ?? [];
  return {
    isSuperRole: roles.includes(SUPER_ROLE),
  };
}
