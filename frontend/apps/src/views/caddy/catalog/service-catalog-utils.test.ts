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
  sitePrimaryHost,
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

  it('buildSiteMetricsMap 补齐无日志 host 为 0', () => {
    const map = buildSiteMetricsMap(
      [{ host: 'a.example.com', count4xx: 2, count5xx: 1 }],
      ['a.example.com', 'b.example.com'],
    );
    expect(map.get('a.example.com')).toEqual({ count4xx: 2, count5xx: 1 });
    expect(map.get('b.example.com')).toEqual({ count4xx: 0, count5xx: 0 });
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
