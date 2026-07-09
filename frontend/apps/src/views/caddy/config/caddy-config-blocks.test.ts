import { describe, expect, it } from 'vitest';

import { buildCaddyfileFromBlocks, parseCaddyfileToBlocks } from './caddy-config-blocks';
import { buildCaddyfile, validateHealthCheckFields } from './caddy-config-parser';
import { validateStructuredConfig } from './caddy-config-validator';
import type { CaddyFormModel, Handle, HealthCheck, Site } from './types';

function emptyMatch() {
  return {
    host: [] as string[],
    path: [] as string[],
    method: [] as string[],
    header: [] as { key: string; value: string }[],
    query: [] as { key: string; value: string }[],
    expression: ''
  };
}

function modelWithReverseProxy(opts: {
  upstream?: string;
  healthCheck?: HealthCheck;
  lbPolicy?: Handle['lbPolicy'];
  domain?: string;
}): CaddyFormModel {
  const domain = opts.domain ?? 'app.example.com';
  const handle: Handle = {
    id: 'h1',
    type: 'reverse_proxy',
    enabled: true,
    upstream: opts.upstream ?? '127.0.0.1:8080',
    lbPolicy: opts.lbPolicy,
    healthCheck: opts.healthCheck,
    transportProtocol: '',
    tlsInsecureSkipVerify: false
  };
  const site: Site = {
    id: 's1',
    name: domain,
    enabled: true,
    domains: [domain],
    tls: { mode: 'auto' },
    imports: [],
    geoip2Vars: [],
    encode: [],
    routes: [
      {
        id: 'r1',
        name: '默认路由',
        enabled: true,
        match: emptyMatch(),
        handles: [handle],
        logAppend: []
      }
    ]
  };
  return {
    schemaVersion: 1,
    global: {},
    upstreams: [],
    sites: [site]
  };
}

/**
 * Task 10 / 10.1 回归：
 * - Preserved / Complex 合并保真（Req 6.1）
 * - 非法 health path 中文错误（Req 3.5）
 * - 前后端共享 health 契约用例（Req 3.6）——与后端 TestSharedHealthContractCases 同输入
 * **Validates: Requirements 6.1, 6.2, 3.5, 3.6**
 */
describe('service-status-discovery regression: preserved + health', () => {
  it('Preserved/Complex 原文在 Simple 扩展 merge 后保持不变', () => {
    // complex：多 handle + import → preserved
    // simple：reverse_proxy + health → 可编辑
    const source = `
(common_headers) {
  header X-Test preserved-marker-token
}

simple.example.com {
  reverse_proxy 127.0.0.1:8080 {
    lb_policy least_conn
    health_uri /healthz
    health_interval 10s
    health_timeout 2s
  }
}

complex.example.com {
  import common_headers
  handle /api/* {
    reverse_proxy 10.0.0.1:9000
  }
  handle {
    reverse_proxy 10.0.0.2:9000
  }
}
`.trim();

    const draft = parseCaddyfileToBlocks(source);

    // complex 与 snippet 进入 preserved
    const preservedRaws = (draft.preservedBlocks ?? []).map(b => b.raw);
    expect(preservedRaws.some(r => r.includes('header X-Test preserved-marker-token'))).toBe(true);
    expect(preservedRaws.some(r => r.includes('complex.example.com'))).toBe(true);
    // simple 可编辑
    expect(draft.sites.some(s => s.domains.includes('simple.example.com'))).toBe(true);

    const complexPreservedBefore = draft.preservedBlocks
      .filter(b => b.raw.includes('complex.example.com'))
      .map(b => b.raw);
    const snippetBefore = draft.preservedBlocks
      .filter(b => b.raw.includes('preserved-marker-token'))
      .map(b => b.raw);

    // 编辑 simple 站点 health（模拟 Simple 扩展后 merge）
    const simple = draft.sites.find(s => s.domains.includes('simple.example.com'));
    expect(simple).toBeTruthy();
    const rp = simple!.routes[0]?.handles.find(h => h.type === 'reverse_proxy');
    expect(rp).toBeTruthy();
    rp!.healthCheck = { path: '/alive', interval: '5s', timeout: '1s' };
    rp!.lbPolicy = 'ip_hash';

    const merged = buildCaddyfileFromBlocks(draft, { sourceOrder: source });

    // Simple 变更生效
    expect(merged).toContain('health_uri /alive');
    expect(merged).toContain('lb_policy ip_hash');

    // Preserved 原文不得被删改
    for (const raw of complexPreservedBefore) {
      expect(merged).toContain(raw.trim());
    }
    for (const raw of snippetBefore) {
      expect(merged).toContain(raw.trim());
    }
    expect(merged).toContain('header X-Test preserved-marker-token');
    expect(merged).toContain('complex.example.com');
    // complex 多 handle 原文特征
    expect(merged).toContain('handle /api/*');
    expect(merged).toContain('10.0.0.1:9000');
  });

  it('非法 path 不以 / 开头返回中文字段错误', () => {
    const errs = validateHealthCheckFields({ path: 'healthz', interval: '10s' }, '健康检查');
    expect(errs.some(e => e.includes('路径') && e.includes('/'))).toBe(true);
    expect(errs.some(e => /[\u4e00-\u9fff]/.test(e))).toBe(true);

    const model = modelWithReverseProxy({
      healthCheck: { path: 'no-slash', interval: '1s' }
    });
    const structured = validateStructuredConfig(model);
    expect(structured.some(e => e.includes('健康检查') && e.includes('/'))).toBe(true);
  });

  /**
   * Shared contract cases — 与后端 TestSharedHealthContractCases 同输入同指令。
   * 输入: HealthCheck + lb → 期望 Caddyfile 含 health_uri / health_interval / health_timeout / lb_policy
   */
  it.each([
    {
      name: 'full health + least_conn',
      healthCheck: { path: '/healthz', interval: '10s', timeout: '2s' } as HealthCheck,
      lbPolicy: 'least_conn' as const,
      want: ['lb_policy least_conn', 'health_uri /healthz', 'health_interval 10s', 'health_timeout 2s']
    },
    {
      name: 'path only + ip_hash',
      healthCheck: { path: '/ready' } as HealthCheck,
      lbPolicy: 'ip_hash' as const,
      want: ['lb_policy ip_hash', 'health_uri /ready'],
      notWant: ['health_interval', 'health_timeout']
    }
  ])('shared FE/BE health contract: $name', ({ healthCheck, lbPolicy, want, notWant }) => {
    const model = modelWithReverseProxy({ healthCheck, lbPolicy });
    const out = buildCaddyfile(model, { includeGlobal: false });
    for (const directive of want) {
      expect(out).toContain(directive);
    }
    for (const directive of notWant ?? []) {
      expect(out).not.toMatch(new RegExp(`\\b${directive}\\b`));
    }
  });
});
