export const routeTitleMap: Record<string, string> = {
  caddy: 'Caddy 配置',
  caddy_access: '访问控制',
  caddy_config: '配置管理',
  caddy_log: '访问日志',
  caddy_source: '来源管理',
  'caddy_system-log': '系统日志',
  caddy_system_log: '系统日志',
  cron: '定时任务',
  dashboard: '仪表盘',
  manage: '系统管理',
  manage_menu: '菜单管理',
  manage_role: '角色管理',
  manage_user: '用户管理',
  notification: '通知管理',
  notification_channel: '通知渠道',
  notification_log: '发送日志',
  notification_rule: '通知规则',
  notification_template: '通知模板',
  security: '安全管理',
  security_binding: '绑定管理',
  security_crs: 'CRS 调优',
  security_exclusion: '排除规则',
  security_job: '任务管理',
  security_observe: '观测日志',
  security_ops: '运维操作',
  security_policy: '策略管理',
  security_release: '发布管理',
  security_runtime: '运行时',
  security_source: '来源管理',
  user: '个人中心',
};

export function findRouteTitle(...keys: Array<null | string | undefined>) {
  for (const key of keys) {
    const titleKey = key?.trim();
    if (!titleKey) continue;

    const normalizedKey = titleKey.replace(/^route\./, '');
    const title = routeTitleMap[titleKey] || routeTitleMap[normalizedKey];
    if (title) return title;
  }

  return '';
}
