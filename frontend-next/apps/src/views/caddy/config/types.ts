export type CaddyFormModel = {
  schemaVersion?: number;
  global: GlobalConfig;
  upstreams: Upstream[];
  sites: Site[];
};

export type GlobalConfig = {
  tls?: TLSConfig;
  headers?: HeaderRule[];
  gzip?: boolean;
  rateLimit?: RateLimit;
  raw?: string;
};

export type Site = {
  id: string;
  name: string;
  enabled: boolean;
  domains: string[];
  tls?: TLSConfig;
  imports?: string[];
  geoip2Vars?: string[];
  encode?: string[];
  headers?: HeaderRule[];
  routes: Route[];
};

export type Route = {
  id: string;
  name: string;
  enabled: boolean;
  match: RouteMatch;
  handles: Handle[];
  logAppend?: KeyValue[];
};

export type RouteMatch = {
  host: string[];
  path: string[];
  method: string[];
  header: KeyValue[];
  query: KeyValue[];
  expression?: string;
};

export type HandleType = 'reverse_proxy' | 'file_server' | 'respond' | 'redirect' | 'header' | 'rewrite';

export type Handle = {
  id: string;
  type: HandleType;
  enabled: boolean;
  upstream?: string;
  lbPolicy?: 'round_robin' | 'least_conn' | 'ip_hash';
  healthCheck?: HealthCheck;
  transportProtocol?: string;
  tlsInsecureSkipVerify?: boolean;
  root?: string;
  browse?: boolean;
  status?: number;
  body?: string;
  to?: string;
  code?: number;
  rules?: HeaderRule[];
  uri?: string;
};

export type Upstream = {
  name: string;
  targets: string[];
  lbPolicy?: 'round_robin' | 'least_conn' | 'ip_hash';
  healthCheck?: HealthCheck;
};

export type TLSConfig = {
  mode?: 'auto' | 'manual' | 'off' | 'internal';
  certFile?: string;
  keyFile?: string;
};

export type HeaderRule = {
  op: 'set' | 'add' | 'delete';
  key: string;
  value?: string;
};

export type RateLimit = {
  zone: string;
  rate: string;
  burst?: number;
};

export type HealthCheck = {
  path: string;
  interval?: string;
  timeout?: string;
};

export type KeyValue = {
  key: string;
  value: string;
};

/** 分块工作台模式（用于保存接口 mode 字段） */
export type CaddyConfigBlockMode = 'blocks' | 'raw';

/** 页面展示模式（UI 切换用，比 BlockMode 多出 waf / preview） */
export type CaddyPageMode = CaddyConfigBlockMode | 'waf' | 'preview';

/** 无法结构化编辑的 Caddyfile 块，只读保留 */
export interface PreservedCaddyBlock {
  id: string;
  kind: 'global' | 'snippet' | 'site' | 'unknown';
  title: string;
  raw: string;
  reason: string;
}

/** 分块草稿，包含可编辑表单和只读保留块 */
export type CaddyBlockDraft = CaddyFormModel & {
  preservedBlocks: PreservedCaddyBlock[];
};
