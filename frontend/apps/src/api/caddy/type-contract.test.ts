import * as fc from 'fast-check';
import { describe, expect, it } from 'vitest';

import { listOf, totalOf, type ListResult } from '../_utils';

import type { CaddyServerApi } from './server';
import type { CaddyIpRegionApi } from './ip-region';
import type { CaddyWafIntegrationApi } from './integration';
import type { CaddySimpleWafApi } from './simple-waf';
import type { CaddyWafSourceApi } from './source';

/** 恒等映射 */
function identityMap<T>(value: T): T {
  return value;
}

/** 深拷贝载荷 */
function clonePayload<T>(value: T): T {
  return structuredClone(value);
}

// 样例响应

const sampleCaddyServer: CaddyServerApi.CaddyServer = {
  id: 1,
  name: 'edge-1',
  url: 'http://127.0.0.1:2019',
  type: 'local',
  createdAt: '2024-01-01T00:00:00Z',
};

const sampleCaddyConfig: CaddyServerApi.CaddyConfig = {
  config: '{\n  auto_https off\n}\n',
  modules: '{"sites":[]}',
};

const sampleConfigPreview: CaddyServerApi.ConfigPreview = {
  valid: true,
  config: 'example.com {\n  reverse_proxy 127.0.0.1:8080\n}\n',
  errors: [],
  actions: ['reload'],
};

const sampleConfigHistoryItem: CaddyServerApi.ConfigHistoryItem = {
  id: 10,
  serverId: 1,
  action: 'save',
  hash: 'abc123',
  createdAt: '2024-02-01T12:00:00Z',
};

const sampleSimpleWafConfig: CaddySimpleWafApi.SimpleWafConfig = {
  serverId: 1,
  enabled: true,
  integrated: false,
  mode: 'detectiononly',
  strength: 'balanced',
  audit: 'relevantonly',
  requestBodyAccess: true,
  requestBodyLimit: 1_310_720,
  requestBodyNoFilesLimit: 131_072,
  siteAddresses: ['example.com'],
  availableSites: ['example.com', 'api.example.com'],
  corazaVersion: '3.2.1',
  crsVersion: '4.0.0',
  actions: ['preview'],
  directives: 'SecRuleEngine DetectionOnly',
  config: '',
  message: 'ok',
};

const sampleWafSource: CaddyWafSourceApi.WafSource = {
  id: 3,
  name: 'crs-official',
  kind: 'crs',
  mode: 'remote',
  url: 'https://example.com/crs.tgz',
  checksumUrl: 'https://example.com/crs.tgz.sha256',
  proxyUrl: '',
  authType: 'none',
  schedule: '0 3 * * *',
  enabled: true,
  autoCheck: true,
  autoDownload: false,
  autoActivate: false,
  lastRelease: 'v4.0.0',
  lastError: '',
  createdAt: '2024-03-01T00:00:00Z',
  updatedAt: '2024-03-02T00:00:00Z',
};

const sampleWafEngineStatus: CaddyWafSourceApi.WafEngineStatus = {
  serverId: 1,
  currentVersion: '3.2.1',
  latestVersion: '3.3.0',
  canUpgrade: true,
  checkedAt: '2024-03-03T08:00:00Z',
  source: 'crs-official',
  message: '',
};

const sampleCaddyLog: CaddyServerApi.CaddyLogItem = {
  id: 42,
  logTime: '2024-03-04T08:00:00Z',
  country: '中国',
  province: '广东省',
  city: '深圳市',
  location: '中国 广东省 深圳市',
  host: 'example.com',
  method: 'GET',
  uri: '/health',
  status: 200,
  size: 1234,
  remoteIp: '192.0.2.1',
  clientIp: '198.51.100.2',
  userAgent: 'LogFlux test',
  rawLog: '{"request":{"uri":"/health"}}',
};

const sampleCaddyLogQuery: CaddyServerApi.CaddyLogQuery = {
  page: 1,
  pageSize: 20,
  host: 'example.com',
  status: -1,
  sortBy: 'logTime',
  order: 'desc',
};

const sampleIpRegionConfig: CaddyIpRegionApi.IpRegionConfig = {
  enabled: true,
  allowList: ['中国', '日本'],
};

const sampleIntegrationStatus: CaddyWafIntegrationApi.IntegrationStatus = {
  serverId: 1,
  integrated: true,
  orderReady: true,
  snippetReady: true,
  directiveReady: true,
  importedSites: ['example.com'],
  availableSites: ['example.com', 'api.example.com'],
  message: '集成状态正常',
};

const sampleIntegrationApplyResult: CaddyWafIntegrationApi.IntegrationApplyResult = {
  serverId: 1,
  enabled: true,
  changed: true,
  importedSites: ['example.com'],
  actions: ['reload'],
  config: 'example.com {\n  waf\n}',
  message: '应用成功',
};

const sampleServerListResult: CaddyServerApi.CaddyServerListResult = {
  list: [sampleCaddyServer],
};

const sampleHistoryListResult: CaddyServerApi.ConfigHistoryListResult = {
  list: [sampleConfigHistoryItem],
  total: 1,
};

const sampleWafSourceListResult: CaddyWafSourceApi.WafSourceListResult = {
  list: [sampleWafSource],
  total: 1,
};

// ---------------------------------------------------------------------------
// Arbitraries — 生成与收紧类型字段一致的运行时对象
// ---------------------------------------------------------------------------

const caddyServerArb: fc.Arbitrary<CaddyServerApi.CaddyServer> = fc.record({
  id: fc.integer({ min: 1, max: 1_000_000 }),
  name: fc.string({ minLength: 1, maxLength: 32 }),
  url: fc.webUrl(),
  type: fc.constantFrom('local', 'remote'),
  createdAt: fc.constantFrom(
    '2024-01-01T00:00:00Z',
    '2025-06-15T12:30:00Z',
  ),
});

const configHistoryItemArb: fc.Arbitrary<CaddyServerApi.ConfigHistoryItem> =
  fc.record({
    id: fc.integer({ min: 1, max: 1_000_000 }),
    serverId: fc.integer({ min: 1, max: 1_000_000 }),
    action: fc.constantFrom('save', 'rollback', 'apply'),
    hash: fc.string({ minLength: 4, maxLength: 40 }),
    createdAt: fc.constantFrom(
      '2024-01-01T00:00:00Z',
      '2025-06-15T12:30:00Z',
    ),
  });

const wafSourceArb: fc.Arbitrary<CaddyWafSourceApi.WafSource> = fc.record({
  id: fc.integer({ min: 1, max: 1_000_000 }),
  name: fc.string({ minLength: 1, maxLength: 32 }),
  kind: fc.constantFrom('crs', 'coraza_engine'),
  mode: fc.constantFrom('remote', 'manual'),
  url: fc.webUrl(),
  checksumUrl: fc.webUrl(),
  proxyUrl: fc.option(fc.webUrl(), { nil: undefined }),
  authType: fc.constantFrom('none', 'token', 'basic'),
  schedule: fc.constantFrom('', '0 3 * * *', '@daily'),
  enabled: fc.boolean(),
  autoCheck: fc.boolean(),
  autoDownload: fc.boolean(),
  autoActivate: fc.boolean(),
  lastRelease: fc.string({ maxLength: 24 }),
  lastError: fc.string({ maxLength: 64 }),
  createdAt: fc.constantFrom('2024-01-01T00:00:00Z'),
  updatedAt: fc.constantFrom('2024-01-02T00:00:00Z'),
});

const simpleWafConfigArb: fc.Arbitrary<CaddySimpleWafApi.SimpleWafConfig> =
  fc.record({
    serverId: fc.integer({ min: 0, max: 1_000_000 }),
    enabled: fc.boolean(),
    integrated: fc.boolean(),
    mode: fc.constantFrom('detectiononly', 'off', 'on'),
    strength: fc.constantFrom('balanced', 'high_blocking', 'low_fp'),
    audit: fc.constantFrom('off', 'on', 'relevantonly'),
    requestBodyAccess: fc.boolean(),
    requestBodyLimit: fc.integer({ min: 0, max: 10_000_000 }),
    requestBodyNoFilesLimit: fc.integer({ min: 0, max: 1_000_000 }),
    siteAddresses: fc.array(fc.domain(), { maxLength: 5 }),
    availableSites: fc.array(fc.domain(), { maxLength: 8 }),
    corazaVersion: fc.option(fc.string({ maxLength: 16 }), { nil: undefined }),
    crsVersion: fc.option(fc.string({ maxLength: 16 }), { nil: undefined }),
    actions: fc.option(fc.array(fc.string({ maxLength: 12 }), { maxLength: 4 }), {
      nil: undefined,
    }),
    directives: fc.option(fc.string({ maxLength: 64 }), { nil: undefined }),
    config: fc.option(fc.string({ maxLength: 64 }), { nil: undefined }),
    message: fc.option(fc.string({ maxLength: 64 }), { nil: undefined }),
  });

const wafEngineStatusArb: fc.Arbitrary<CaddyWafSourceApi.WafEngineStatus> =
  fc.record({
    serverId: fc.integer({ min: 0, max: 1_000_000 }),
    currentVersion: fc.option(fc.string({ maxLength: 16 }), { nil: undefined }),
    latestVersion: fc.option(fc.string({ maxLength: 16 }), { nil: undefined }),
    canUpgrade: fc.boolean(),
    checkedAt: fc.option(fc.constantFrom('2024-01-01T00:00:00Z'), {
      nil: undefined,
    }),
    source: fc.option(fc.string({ maxLength: 24 }), { nil: undefined }),
    message: fc.option(fc.string({ maxLength: 64 }), { nil: undefined }),
  });

const caddyConfigArb: fc.Arbitrary<CaddyServerApi.CaddyConfig> = fc.record({
  config: fc.string({ maxLength: 200 }),
  modules: fc.option(fc.string({ maxLength: 100 }), { nil: undefined }),
});

const configPreviewArb: fc.Arbitrary<CaddyServerApi.ConfigPreview> = fc.record({
  valid: fc.boolean(),
  config: fc.string({ maxLength: 200 }),
  errors: fc.array(fc.string({ maxLength: 40 }), { maxLength: 5 }),
  actions: fc.array(fc.string({ maxLength: 20 }), { maxLength: 5 }),
});

describe('API_Contract type tightening — runtime payload equivalence', () => {
    it('Property 5: listOf 对列表响应保持元素深度等价', () => {
    fc.assert(
      fc.property(
        fc.array(caddyServerArb, { maxLength: 20 }),
        fc.option(fc.integer({ min: 0, max: 10_000 }), { nil: undefined }),
        (list, total) => {
          const envelope: ListResult<CaddyServerApi.CaddyServer> = {
            list,
            ...(total === undefined ? {} : { total }),
          };
          const mapped = listOf(envelope);
          expect(mapped).toEqual(list);
          // 逐项深度等价，确保未丢业务字段
          for (let i = 0; i < list.length; i++) {
            expect(mapped[i]).toEqual(list[i]);
            expect(mapped[i]).toEqual(identityMap(list[i]));
          }
          expect(totalOf(envelope)).toBe(total ?? 0);
        },
      ),
      { numRuns: 100 },
    );
  });

  it('Property 5: listOf 对裸数组响应保持深度等价', () => {
    fc.assert(
      fc.property(fc.array(wafSourceArb, { maxLength: 15 }), (items) => {
        const mapped = listOf(items);
        expect(mapped).toEqual(items);
        expect(totalOf(items)).toBe(items.length);
      }),
      { numRuns: 100 },
    );
  });

  it('Property 5: 核心模型恒等/克隆映射后业务字段深度等价', () => {
    fc.assert(
      fc.property(
        caddyServerArb,
        caddyConfigArb,
        configPreviewArb,
        configHistoryItemArb,
        simpleWafConfigArb,
        wafSourceArb,
        wafEngineStatusArb,
        (
          server,
          config,
          preview,
          history,
          waf,
          source,
          engine,
        ) => {
          expect(identityMap(server)).toEqual(server);
          expect(clonePayload(server)).toEqual(server);

          expect(identityMap(config)).toEqual(config);
          expect(clonePayload(config)).toEqual(config);

          expect(identityMap(preview)).toEqual(preview);
          expect(clonePayload(preview)).toEqual(preview);

          expect(identityMap(history)).toEqual(history);
          expect(clonePayload(history)).toEqual(history);

          expect(identityMap(waf)).toEqual(waf);
          expect(clonePayload(waf)).toEqual(waf);

          expect(identityMap(source)).toEqual(source);
          expect(clonePayload(source)).toEqual(source);

          expect(identityMap(engine)).toEqual(engine);
          expect(clonePayload(engine)).toEqual(engine);
        },
      ),
      { numRuns: 100 },
    );
  });

  // --- 既有成功响应样例（可读示例，对齐 Req 5.1 主类型清单） ---

  it('样例：CaddyServer 列表 listOf 后字段深度等价', () => {
    const mapped = listOf(sampleServerListResult);
    expect(mapped).toHaveLength(1);
    expect(mapped[0]).toEqual(sampleCaddyServer);
    expect(mapped[0]).toMatchObject({
      id: 1,
      name: 'edge-1',
      url: 'http://127.0.0.1:2019',
      type: 'local',
      createdAt: '2024-01-01T00:00:00Z',
    });
  });

  it('样例：CaddyConfig / ConfigPreview / ConfigHistoryItem 恒等映射', () => {
    expect(identityMap(sampleCaddyConfig)).toEqual(sampleCaddyConfig);
    expect(clonePayload(sampleConfigPreview)).toEqual(sampleConfigPreview);

    const historyMapped = listOf(sampleHistoryListResult);
    expect(historyMapped).toEqual([sampleConfigHistoryItem]);
    expect(totalOf(sampleHistoryListResult)).toBe(1);
    expect(historyMapped[0]).toEqual({
      id: 10,
      serverId: 1,
      action: 'save',
      hash: 'abc123',
      createdAt: '2024-02-01T12:00:00Z',
    });
  });

  it('样例：SimpleWafConfig 映射后不丢业务字段', () => {
    const mapped = identityMap(sampleSimpleWafConfig);
    expect(mapped).toEqual(sampleSimpleWafConfig);
    // 显式核对关键业务字段，防止类型收紧时静默丢字段
    expect(mapped).toMatchObject({
      serverId: 1,
      enabled: true,
      integrated: false,
      mode: 'detectiononly',
      strength: 'balanced',
      audit: 'relevantonly',
      requestBodyAccess: true,
      requestBodyLimit: 1_310_720,
      requestBodyNoFilesLimit: 131_072,
      siteAddresses: ['example.com'],
      availableSites: ['example.com', 'api.example.com'],
      corazaVersion: '3.2.1',
      crsVersion: '4.0.0',
    });
  });

  it('样例：WafSource 列表与 WafEngineStatus 深度等价', () => {
    const sources = listOf(sampleWafSourceListResult);
    expect(sources).toEqual([sampleWafSource]);
    expect(totalOf(sampleWafSourceListResult)).toBe(1);
    expect(sources[0]).toMatchObject({
      id: 3,
      name: 'crs-official',
      kind: 'crs',
      mode: 'remote',
      authType: 'none',
      enabled: true,
      autoCheck: true,
      autoDownload: false,
      autoActivate: false,
    });

    expect(identityMap(sampleWafEngineStatus)).toEqual(sampleWafEngineStatus);
    expect(clonePayload(sampleWafEngineStatus)).toMatchObject({
      serverId: 1,
      currentVersion: '3.2.1',
      latestVersion: '3.3.0',
      canUpgrade: true,
    });
  });

  it('样例：访问日志、IP 地域配置与 WAF 集成响应保持字段完整', () => {
    const logPage: CaddyServerApi.CaddyLogPageResult = {
      list: [sampleCaddyLog],
      total: 1,
    };

    expect(listOf(logPage)).toEqual([sampleCaddyLog]);
    expect(totalOf(logPage)).toBe(1);
    expect(identityMap(sampleCaddyLogQuery)).toEqual(sampleCaddyLogQuery);
    expect(clonePayload(sampleIpRegionConfig)).toEqual(sampleIpRegionConfig);
    expect(identityMap(sampleIntegrationStatus)).toEqual(sampleIntegrationStatus);
    expect(clonePayload(sampleIntegrationApplyResult)).toEqual(
      sampleIntegrationApplyResult,
    );
  });

  it('边界：listOf/totalOf 对 null/undefined/非法 list 不抛且可预期', () => {
    expect(listOf(null)).toEqual([]);
    expect(listOf(undefined)).toEqual([]);
    expect(listOf({ list: undefined as unknown as never[] })).toEqual([]);
    expect(listOf({ list: null as unknown as never[] })).toEqual([]);
    expect(totalOf(null)).toBe(0);
    expect(totalOf(undefined)).toBe(0);
    // empty object is not a valid ListResult at the type level; cast for runtime boundary
    expect(totalOf({} as ListResult<unknown>)).toBe(0);
    expect(totalOf({ list: [1, 2], total: undefined })).toBe(0);
  });
});
