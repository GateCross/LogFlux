export interface PermissionOption {
  label: string;
  value: string;
}

export interface PermissionGroup {
  label: string;
  options: PermissionOption[];
}

export interface PermissionSummary {
  details: string[];
  group: string;
  label: string;
}

/** 前端不展示/不可勾选的历史或内部权限值 */
export const HIDDEN_PERMISSION_VALUES = new Set(['caddy_source', 'user_center']);

export const PERMISSION_GROUPS: PermissionGroup[] = [
  {
    label: '仪表盘',
    options: [{ label: '仪表盘', value: 'dashboard' }],
  },
  {
    label: 'Caddy 配置',
    options: [
      { label: 'Caddy 菜单', value: 'caddy' },
      { label: '配置管理', value: 'caddy_config' },
      { label: '访问控制', value: 'caddy_access' },
      { label: '访问日志', value: 'logs_caddy' },
    ],
  },
  {
    label: '日志与审计',
    options: [{ label: '系统日志', value: 'logs' }],
  },
  {
    label: '定时任务',
    options: [{ label: '定时任务', value: 'cron' }],
  },
  {
    label: '系统管理',
    options: [
      { label: '系统管理', value: 'manage' },
      { label: '用户管理', value: 'manage_user' },
      { label: '角色管理', value: 'manage_role' },
      { label: '菜单管理', value: 'manage_menu' },
    ],
  },
  {
    label: '通知管理',
    options: [
      { label: '通知管理', value: 'notification' },
      { label: '通知渠道', value: 'notification_channel' },
      { label: '通知规则', value: 'notification_rule' },
      { label: '通知模板', value: 'notification_template' },
      { label: '发送日志', value: 'notification_log' },
    ],
  },
];

export const PERMISSION_OPTION_MAP = new Map(
  PERMISSION_GROUPS.flatMap((group) =>
    group.options.map(
      (option) => [option.value, { ...option, group: group.label }] as const,
    ),
  ),
);

/** 去重、去空、过滤隐藏权限 */
export function uniquePermissions(permissions?: string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const permission of permissions ?? []) {
    const value = String(permission || '').trim();
    if (!value || seen.has(value) || HIDDEN_PERMISSION_VALUES.has(value)) {
      continue;
    }
    seen.add(value);
    result.push(value);
  }
  return result;
}

export function permissionLabelOf(permission: string): string {
  return PERMISSION_OPTION_MAP.get(permission)?.label || permission;
}

/** 弹层分组：内置组 + 后端多出的未知权限 */
export function buildVisiblePermissionGroups(
  currentPermissions: string[],
): PermissionGroup[] {
  const extraOptions = uniquePermissions(currentPermissions)
    .filter((permission) => !PERMISSION_OPTION_MAP.has(permission))
    .map((permission) => ({
      label: permission,
      value: permission,
    }));

  if (extraOptions.length === 0) {
    return PERMISSION_GROUPS;
  }

  return [
    ...PERMISSION_GROUPS,
    {
      label: '其他权限',
      options: extraOptions,
    },
  ];
}

export function buildPermissionSummaries(
  permissions?: string[],
): PermissionSummary[] {
  const selected = new Set(uniquePermissions(permissions));
  const summaries: PermissionSummary[] = [];

  for (const group of PERMISSION_GROUPS) {
    const details = group.options
      .filter((option) => selected.has(option.value))
      .map((option) => option.label);
    if (details.length === 0) continue;

    summaries.push({
      details,
      group: group.label,
      label:
        details.length === 1
          ? details[0]!
          : `${group.label} ${details.length}项`,
    });
  }

  const extraDetails = [...selected]
    .filter((permission) => !PERMISSION_OPTION_MAP.has(permission))
    .map(permissionLabelOf);

  if (extraDetails.length > 0) {
    summaries.push({
      details: extraDetails,
      group: '其他权限',
      label:
        extraDetails.length === 1
          ? extraDetails[0]!
          : `其他权限 ${extraDetails.length}项`,
    });
  }

  return summaries;
}

export function visiblePermissionSummaries(
  permissions?: string[],
  limit = 3,
): PermissionSummary[] {
  return buildPermissionSummaries(permissions).slice(0, limit);
}

export function hiddenPermissionSummaries(
  permissions?: string[],
  limit = 3,
): PermissionSummary[] {
  return buildPermissionSummaries(permissions).slice(limit);
}

export function hasVisiblePermissions(permissions?: string[]): boolean {
  return uniquePermissions(permissions).length > 0;
}
