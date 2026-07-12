/** query key 工厂：domain → resource → 参数，业务侧统一走 qk */
export const qk = {
  dashboard: {
    all: ['dashboard'] as const,
    summary: (params?: object) =>
      params == null
        ? ([...qk.dashboard.all, 'summary'] as const)
        : ([...qk.dashboard.all, 'summary', params] as const),
  },
  caddy: {
    servers: () => ['caddy', 'servers'] as const,
    config: (serverId: number) => ['caddy', 'config', serverId] as const,
    history: (serverId: number, page: number, pageSize: number) =>
      ['caddy', 'history', serverId, page, pageSize] as const,
    catalog: (serverId: number) => ['caddy', 'catalog', serverId] as const,
    logs: (params: object) => ['caddy', 'logs', params] as const,
    access: (params?: object) => ['caddy', 'access', params] as const,
    ipRegion: (params?: object) => ['caddy', 'ip-region', params] as const,
    source: (params?: object) => ['caddy', 'source', params] as const,
    wafEngine: () => ['caddy', 'waf-engine'] as const,
    metrics: (serverId: number, siteKey?: string) =>
      ['caddy', 'metrics', serverId, siteKey] as const,
    systemLog: (params: object) => ['caddy', 'system-log', params] as const,
  },
  cron: {
    list: (params: object) => ['cron', 'list', params] as const,
    detail: (id: number) => ['cron', 'detail', id] as const,
  },
  notification: {
    channels: (params?: object) =>
      ['notification', 'channels', params] as const,
    rules: (params?: object) => ['notification', 'rules', params] as const,
    templates: (params?: object) =>
      ['notification', 'templates', params] as const,
    logs: (params?: object) => ['notification', 'logs', params] as const,
    events: () => ['notification', 'events'] as const,
  },
  system: {
    users: (params: object) => ['system', 'users', params] as const,
    roles: (params?: object) => ['system', 'roles', params] as const,
    menus: () => ['system', 'menus'] as const,
    systemLogs: (params: object) => ['system', 'logs', params] as const,
  },
  user: {
    center: () => ['user', 'center'] as const,
  },
} as const;
