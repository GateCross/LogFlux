import { describe, expect, it } from 'vitest';

import type { CaddyFormModel } from '../config/types';
import {
  buildAccessLogQuery,
  buildCatalogSitesFromConfigPayload,
  buildCatalogSitesFromModel,
  buildConfigWorkbenchQuery,
  buildSiteMetricsMap,
  collectCatalogHosts,
  formatMetricCount,
  metricCountColor,
  normalizeSiteHost,
  siteCertDaysLeft,
  siteMetricsForDomains,
  sitePrimaryHost,
  siteProtectBadge,
  statusLabel,
  statusTagColor,
} from './service-catalog-utils';

function sampleModel(): CaddyFormModel {
  return {
    schemaVersion: 1,
    global: { raw: '' },
    upstreams: [],
    sites: [
      {
        id: 's1',
        name: '官网',
        enabled: true,
        domains: ['www.example.com', ':443'],
        tls: { mode: 'auto' },
        imports: [],
        geoip2Vars: [],
        encode: [],
        routes: [
          {
            id: 'r1',
            name: '默认路由',
            enabled: true,
            match: {
              host: [],
              path: [],
              method: [],
              header: [],
              query: [],
              expression: '',
            },
            handles: [
              {
                id: 'h1',
                type: 'reverse_proxy',
                enabled: true,
                upstream: '127.0.0.1:8080',
                lbPolicy: 'least_conn',
                healthCheck: { path: '/health', interval: '10s', timeout: '5s' },
              },
            ],
            logAppend: [],
          },
        ],
      },
      {
        id: 's2',
        name: '复杂站',
        enabled: true,
        domains: ['complex.example.com'],
        tls: { mode: 'auto' },
        imports: ['shared'],
        geoip2Vars: [],
        encode: [],
        routes: [
          {
            id: 'r2',
            name: '默认路由',
            enabled: true,
            match: {
              host: [],
              path: [],
              method: [],
              header: [],
              query: [],
              expression: '',
            },
            handles: [
              {
                id: 'h2',
                type: 'reverse_proxy',
                enabled: true,
                upstream: '127.0.0.1:9000',
              },
            ],
            logAppend: [],
          },
        ],
      },
    ],
  };
}

describe('service-catalog-utils', () => {
  it('normalizeSiteHost 去掉协议/端口/路径，忽略纯监听地址', () => {
    expect(normalizeSiteHost('https://API.Example.com:443/path')).toBe('api.example.com');
    expect(normalizeSiteHost('www.example.com:8080')).toBe('www.example.com');
    expect(normalizeSiteHost(':8080')).toBe('');
    expect(normalizeSiteHost('')).toBe('');
  });

  it('sitePrimaryHost 取第一个有效 host', () => {
    expect(sitePrimaryHost([':8080', 'a.example.com', 'b.example.com'])).toBe('a.example.com');
    expect(sitePrimaryHost([':80'])).toBe('');
  });

  it('buildCatalogSitesFromModel 合并 simple + complex，不复制数据源语义', () => {
    const cards = buildCatalogSitesFromModel(sampleModel());
    expect(cards).toHaveLength(2);

    const simple = cards.find((c) => c.id === 's1');
    expect(simple?.kind).toBe('simple');
    expect(simple?.mode).toBe('reverse_proxy');
    expect(simple?.primaryHost).toBe('www.example.com');
    expect(simple?.upstream).toBe('127.0.0.1:8080');
    expect(simple?.lbPolicy).toBe('least_conn');
    expect(simple?.healthPath).toBe('/health');

    const complex = cards.find((c) => c.id === 's2');
    expect(complex?.kind).toBe('complex');
    expect(complex?.mode).toBe('complex');
    expect(complex?.reasons?.some((r) => r.includes('import'))).toBe(true);
  });

  it('回归：线上 snippet 风格站点（import 能力挂载 + https 上游跳过验证）判 simple', () => {
    // 取自生产 Caddyfile 的 pve 站点块原文（10-20 fix_lan_waf 版本）
    const config = `{
  admin localhost:2019
}

pve.myddpp.top {
  import deny_403_redirect
  import waf_protect
  import cloudflare
  import geo_auth
  handle {
    reverse_proxy https://192.168.50.100:8006 {
      transport http {
        tls_insecure_skip_verify
      }
    }
  }
}
`;
    const cards = buildCatalogSitesFromConfigPayload({ config });
    const pve = cards.find((c) => c.primaryHost === 'pve.myddpp.top');
    expect(pve?.kind).toBe('simple');
    expect(pve?.mode).toBe('reverse_proxy');
    expect(pve?.upstream).toBe('https://192.168.50.100:8006');
  });

  it('buildCatalogSitesFromConfigPayload 从 Caddyfile 解析站点', () => {
    const cards = buildCatalogSitesFromConfigPayload({
      config: `
www.demo.com {
  reverse_proxy 10.0.0.1:8080 {
    lb_policy round_robin
    health_uri /ready
  }
}
`,
    });
    expect(cards.length).toBeGreaterThanOrEqual(1);
    expect(cards.some((c) => c.domains.includes('www.demo.com') || c.primaryHost === 'www.demo.com')).toBe(
      true,
    );
  });

  it('collectCatalogHosts 去重并规范化', () => {
    const hosts = collectCatalogHosts([
      {
        id: '1',
        name: 'a',
        domains: ['A.example.com', ':8080'],
        primaryHost: 'a.example.com',
        kind: 'simple',
        mode: 'reverse_proxy',
        enabled: true,
      },
      {
        id: '2',
        name: 'b',
        domains: ['b.example.com'],
        primaryHost: 'b.example.com',
        kind: 'simple',
        mode: 'file_server',
        enabled: true,
      },
    ]);
    expect(hosts.sort()).toEqual(['a.example.com', 'b.example.com']);
  });

  it('buildSiteMetricsMap 补齐无日志 host 为 0（含今日 total/blocked）', () => {
    const map = buildSiteMetricsMap(
      [{ host: 'a.example.com', count4xx: 2, count5xx: 1 }],
      ['a.example.com', 'b.example.com'],
    );
    expect(map.get('a.example.com')).toEqual({ count4xx: 2, count5xx: 1 });
    expect(map.get('b.example.com')).toEqual({
      count4xx: 0,
      count5xx: 0,
      total: 0,
      blocked: 0,
    });
  });

  it('buildSiteMetricsMap 透传 window=24h 契约字段 total/blocked', () => {
    const map = buildSiteMetricsMap([
      { host: 'a.example.com', count4xx: 2, count5xx: 1, total: 120, blocked: 3 },
      // 后端未上线 total/blocked 时保持 undefined，展示按 — 降级
      { host: 'b.example.com', count4xx: 0, count5xx: 0 },
    ]);
    expect(map.get('a.example.com')).toEqual({
      count4xx: 2,
      count5xx: 1,
      total: 120,
      blocked: 3,
    });
    expect(map.get('b.example.com')?.total).toBeUndefined();
    expect(map.get('b.example.com')?.blocked).toBeUndefined();
  });

  it('siteMetricsForDomains 汇总站点全部域名的流量', () => {
    const map = buildSiteMetricsMap([
      { host: 'a.example.com', count4xx: 2, count5xx: 1, total: 120, blocked: 3 },
      { host: 'b.example.com', count4xx: 5, count5xx: 4, total: 80, blocked: 7 },
    ]);

    // 主域名只有 120 请求：旧实现会漏掉 b.example.com 的 80 请求与 7 次拦截
    const metrics = siteMetricsForDomains(map, ['a.example.com', 'b.example.com']);
    expect(metrics).toEqual({
      count4xx: 7,
      count5xx: 5,
      total: 200,
      blocked: 10,
    });
  });

  it('siteMetricsForDomains 去除端口与协议后仍能匹配', () => {
    const map = buildSiteMetricsMap([
      { host: 'a.example.com', count4xx: 0, count5xx: 0, total: 10, blocked: 0 },
      { host: 'b.example.com', count4xx: 0, count5xx: 0, total: 5, blocked: 0 },
    ]);
    const metrics = siteMetricsForDomains(map, [
      'https://a.example.com',
      'b.example.com:8443',
    ]);
    expect(metrics?.total).toBe(15);
  });

  it('siteMetricsForDomains 任一域名缺值则 total/blocked 整体降级为 undefined', () => {
    const map = buildSiteMetricsMap([
      { host: 'a.example.com', count4xx: 2, count5xx: 1, total: 120, blocked: 3 },
      // 后端只回了一个域名的新字段，另一个没有 → 不做半有半无的小计
      { host: 'b.example.com', count4xx: 1, count5xx: 0 },
    ]);
    const metrics = siteMetricsForDomains(map, ['a.example.com', 'b.example.com']);
    expect(metrics?.total).toBeUndefined();
    expect(metrics?.blocked).toBeUndefined();
    // 4xx/5xx 恒为数值，缺失按 0 累加
    expect(metrics?.count4xx).toBe(3);
    expect(metrics?.count5xx).toBe(1);
  });

  it('siteMetricsForDomains 无可用域名返回 null', () => {
    expect(siteMetricsForDomains(new Map(), [])).toBeNull();
    expect(siteMetricsForDomains(new Map(), [':8080'])).toBeNull();
  });

  it('siteCertDaysLeft 按域名/泛域名匹配并取最小剩余天数', () => {
    const certs = [
      { id: 1, serverId: 1, domains: 'www.example.com,api.example.com', primaryDomain: 'www.example.com', issuer: '', source: 'upload' as const, notBefore: '', notAfter: '', lastSyncedAt: '', storagePath: '', status: 'valid' as const, daysLeft: 30 },
      { id: 2, serverId: 1, domains: '*.example.com', primaryDomain: '*.example.com', issuer: '', source: 'acme_dns' as const, notBefore: '', notAfter: '', lastSyncedAt: '', storagePath: '', status: 'expiring' as const, daysLeft: 7 },
      { id: 3, serverId: 1, domains: 'other.org', primaryDomain: 'other.org', issuer: '', source: 'upload' as const, notBefore: '', notAfter: '', lastSyncedAt: '', storagePath: '', status: 'valid' as const, daysLeft: 90 },
    ];
    // 同时命中精确与泛域名时取最小剩余天数
    expect(siteCertDaysLeft(certs, ['www.example.com', ':443'])).toBe(7);
    // 仅泛域名覆盖
    expect(siteCertDaysLeft(certs, ['sub.example.com'])).toBe(7);
    // 未命中证书
    expect(siteCertDaysLeft(certs, ['unmatched.net'])).toBeNull();
    expect(siteCertDaysLeft([], ['www.example.com'])).toBeNull();
    expect(siteCertDaysLeft(certs, [])).toBeNull();
  });

  it('siteProtectBadge 组合站点挂载与全局引擎状态', () => {
    const mountOn = { capabilities: { waf: { status: 'on' as const } } };
    const mountOff = { capabilities: { waf: { status: 'off' as const } } };
    const mountUnknown = { capabilities: { waf: { status: 'unknown' as const } } };

    expect(siteProtectBadge(mountOff, null)).toMatchObject({ label: '未防护' });
    expect(siteProtectBadge(mountUnknown, null)).toMatchObject({ label: 'WAF?' });
    // 无全局快照：挂载未知引擎状态
    expect(siteProtectBadge(mountOn, null)).toMatchObject({ label: '防护中' });
    // 全局引擎关闭
    expect(siteProtectBadge(mountOn, { enabled: false, mode: 'off' })).toMatchObject({
      label: '未生效',
    });
    // 阻断 / 检测
    expect(siteProtectBadge(mountOn, { enabled: true, mode: 'on' })).toMatchObject({
      label: '阻断',
    });
    expect(
      siteProtectBadge(mountOn, { enabled: true, mode: 'detectiononly' }),
    ).toMatchObject({ label: '检测' });
  });

  it('指标展示格式与颜色', () => {
    expect(formatMetricCount(undefined)).toBe('—');
    expect(formatMetricCount(3)).toBe('3');
    expect(metricCountColor(0, '5xx')).toBe('default');
    expect(metricCountColor(2, '5xx')).toBe('error');
    expect(metricCountColor(2, '4xx')).toBe('warning');
  });

  it('探测状态标签', () => {
    expect(statusLabel(undefined)).toBe('未探测');
    expect(statusLabel({ serverId: 1, name: 'n', reachable: true, latencyMs: 12, probedAt: '' })).toBe(
      '在线',
    );
    expect(statusTagColor({ serverId: 1, name: 'n', reachable: false, latencyMs: 0, probedAt: '' })).toBe(
      'error',
    );
  });

  it('深链 query 复用配置工作台与访问日志，不引入新数据源', () => {
    expect(buildConfigWorkbenchQuery({ serverId: 3, mode: 'waf' })).toEqual({
      serverId: '3',
      mode: 'waf',
    });
    expect(buildAccessLogQuery([':8080', 'log.example.com'])).toEqual({ host: 'log.example.com' });
    expect(buildAccessLogQuery([':8080'])).toEqual({});
  });
});
