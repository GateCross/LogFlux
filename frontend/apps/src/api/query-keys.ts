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
    /** 单实例：配置、目录、证书都只有一份，key 不再带 serverId */
    config: () => ['caddy', 'config'] as const,
    history: (page: number, pageSize: number) =>
      ['caddy', 'history', page, pageSize] as const,
    catalog: () => ['caddy', 'catalog'] as const,
    certificates: () => ['caddy', 'certificates'] as const,
    logs: (params: object) => ['caddy', 'logs', params] as const,
    access: (params?: object) => ['caddy', 'access', params] as const,
    globalOptions: (params?: object) =>
      ['caddy', 'global-options', params] as const,
    entryBlock: (params?: object) =>
      ['caddy', 'entry-block', params] as const,
    ipRegion: (params?: object) => ['caddy', 'ip-region', params] as const,
    source: (params?: object) => ['caddy', 'source', params] as const,
    wafEngine: () => ['caddy', 'waf-engine'] as const,
    wafEvents: (params: object) => ['caddy', 'waf-events', params] as const,
    wafEventStats: (params?: object) =>
      ['caddy', 'waf-event-stats', params] as const,
    /** 站点指标；windowMinutes 传入时进 key（不同窗口缓存隔离，如 24h=1440） */
    metrics: (siteKey?: string, windowMinutes?: number) =>
      windowMinutes == null
        ? (['caddy', 'metrics', siteKey] as const)
        : (['caddy', 'metrics', siteKey, windowMinutes] as const),
    /** 站点自带图标；按 host 隔离，抓空也缓存（避免每次进页重试） */
    siteIcon: (host: string) => ['caddy', 'site-icon', host] as const,
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
    systemLogs: (params: object) => ['system', 'logs', params] as const,
  },
  user: {
    center: () => ['user', 'center'] as const,
  },
} as const;
