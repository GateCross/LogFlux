export type SecurityTabKey = 'source' | 'runtime' | 'crs' | 'exclusion' | 'binding' | 'observe' | 'release' | 'job';

export type SecurityMenuKey = 'source' | 'policy' | 'observe' | 'ops';

export interface SecurityMenuSchemaItem {
  key: SecurityMenuKey;
  routeName: string;
  title: string;
  tabs: SecurityTabKey[];
  defaultTab: SecurityTabKey;
}

export const SECURITY_OBSERVE_QUERY_KEYS = ['policyId', 'window', 'intervalSec', 'topN', 'host', 'path', 'method'] as const;

export const SECURITY_MENU_SCHEMA: Record<SecurityMenuKey, SecurityMenuSchemaItem> = {
  source: {
    key: 'source',
    routeName: 'security_source',
    title: '规则来源',
    tabs: ['source'],
    defaultTab: 'source'
  },
  policy: {
    key: 'policy',
    routeName: 'security_policy',
    title: '策略中心',
    tabs: ['runtime', 'crs', 'exclusion', 'binding'],
    defaultTab: 'runtime'
  },
  observe: {
    key: 'observe',
    routeName: 'security_observe',
    title: '观测与处置',
    tabs: ['observe'],
    defaultTab: 'observe'
  },
  ops: {
    key: 'ops',
    routeName: 'security_ops',
    title: '发布运维',
    tabs: ['release', 'job'],
    defaultTab: 'release'
  }
};

export const SECURITY_TAB_MENU_MAP: Record<SecurityTabKey, SecurityMenuKey> = {
  source: 'source',
  runtime: 'policy',
  crs: 'policy',
  exclusion: 'policy',
  binding: 'policy',
  observe: 'observe',
  release: 'ops',
  job: 'ops'
};

export const SECURITY_ROUTE_NAME_MENU_MAP: Record<string, SecurityMenuKey> = {
  security_source: 'source',
  security_policy: 'policy',
  security_observe: 'observe',
  security_ops: 'ops',
  security_runtime: 'policy',
  security_crs: 'policy',
  security_exclusion: 'policy',
  security_binding: 'policy',
  security_release: 'ops',
  security_job: 'ops'
};

export const SECURITY_TAB_TITLE_MAP: Record<SecurityTabKey, string> = {
  source: '更新源配置',
  runtime: '运行模式',
  crs: 'CRS 调优',
  exclusion: '规则例外',
  binding: '策略绑定',
  observe: '策略观测',
  release: '版本发布管理',
  job: '任务日志'
};

export function pickRouteQueryValue(value: unknown) {
  if (Array.isArray(value)) {
    return String(value[0] ?? '').trim();
  }

  if (value == null) {
    return '';
  }

  return String(value).trim();
}

export function isSecurityTabKey(value: string): value is SecurityTabKey {
  return value in SECURITY_TAB_MENU_MAP;
}

export function resolveSecurityMenuFromRoute(routeName: string, routeQueryActiveTab: unknown) {
  const matched = SECURITY_ROUTE_NAME_MENU_MAP[String(routeName || '')];
  if (matched) {
    return matched;
  }

  const legacyTab = pickRouteQueryValue(routeQueryActiveTab);
  if (isSecurityTabKey(legacyTab)) {
    return SECURITY_TAB_MENU_MAP[legacyTab];
  }

  return 'source';
}

export function resolveSecurityTabFromRoute(menu: SecurityMenuKey, routeQueryActiveTab: unknown) {
  const groupTabs = SECURITY_MENU_SCHEMA[menu]?.tabs || ['source'];
  const routeTab = pickRouteQueryValue(routeQueryActiveTab);
  if (isSecurityTabKey(routeTab) && groupTabs.includes(routeTab)) {
    return routeTab;
  }

  return SECURITY_MENU_SCHEMA[menu]?.defaultTab || groupTabs[0];
}

export function getSecurityMenuTabs(menu: SecurityMenuKey) {
  return SECURITY_MENU_SCHEMA[menu]?.tabs || ['source'];
}

export function getSecurityMenuRouteName(menu: SecurityMenuKey) {
  return SECURITY_MENU_SCHEMA[menu]?.routeName;
}

export function getSecurityMenuByTab(tab: SecurityTabKey) {
  return SECURITY_TAB_MENU_MAP[tab];
}

export function getSecurityDefaultTab(menu: SecurityMenuKey) {
  return SECURITY_MENU_SCHEMA[menu]?.defaultTab || 'source';
}

export function getSecurityTabTitle(tab: SecurityTabKey) {
  return SECURITY_TAB_TITLE_MAP[tab];
}

export function isSecurityTabVisible(menu: SecurityMenuKey, tab: SecurityTabKey) {
  return getSecurityMenuTabs(menu).includes(tab);
}

export function isSecurityMenuTabNavVisible(menu: SecurityMenuKey) {
  return getSecurityMenuTabs(menu).length > 1;
}
