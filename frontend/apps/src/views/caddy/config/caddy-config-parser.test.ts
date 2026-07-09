import * as fc from 'fast-check';
import { describe, expect, it } from 'vitest';

import {
  buildCaddyfile,
  hasEmittableHealthCheck,
  parseCaddyfileToModules,
  validateHealthCheckFields
} from './caddy-config-parser';
import { validateStructuredConfig } from './caddy-config-validator';
import type { CaddyFormModel, Handle, HealthCheck, Site } from './types';

const durationArb = fc.tuple(fc.integer({ min: 1, max: 3600 }), fc.constantFrom('ms', 's', 'm', 'h')).map(
  ([n, unit]) => `${n}${unit}`
);

const healthPathArb = fc
  .string({ minLength: 1, maxLength: 24 })
  .filter(s => /^[a-zA-Z0-9_/-]+$/.test(s) && !s.includes('//'))
  .map(s => `/${s.replace(/^\//, '')}`);

/** 启用且 path 合法的 Health_Check */
const enabledHealthCheckArb: fc.Arbitrary<HealthCheck> = fc.record({
  path: healthPathArb,
  interval: fc.option(durationArb, { nil: undefined }),
  timeout: fc.option(durationArb, { nil: undefined })
});

const lbPolicyArb = fc.constantFrom('round_robin' as const, 'least_conn' as const, 'ip_hash' as const);

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

describe('caddy-config-parser health / lb generation', () => {
  /**
   * Property 3: Health 指令与模型一致
   * For any enabled Health_Check with valid path, generated Caddyfile contains
   * matching health_uri (and health_interval/health_timeout when set);
   * when health is disabled or all-empty, none of the three directives appear.
   * **Validates: Requirements 3.1, 3.2**
   */
  it('Property 3: Health 指令与模型一致', () => {
    fc.assert(
      fc.property(
        fc.oneof(
          // 启用且 path 合法
          fc.record({
            kind: fc.constant('enabled' as const),
            healthCheck: enabledHealthCheckArb,
            lbPolicy: fc.option(lbPolicyArb, { nil: undefined })
          }),
          // 关闭 / 全空：不写 health
          fc.record({
            kind: fc.constant('disabled' as const),
            healthCheck: fc.constantFrom(
              undefined,
              { path: '' },
              { path: '   ' },
              { path: '', interval: '', timeout: '' }
            ),
            lbPolicy: fc.option(lbPolicyArb, { nil: undefined })
          })
        ),
        input => {
          const model = modelWithReverseProxy({
            healthCheck: input.healthCheck as HealthCheck | undefined,
            lbPolicy: input.lbPolicy
          });
          const caddyfile = buildCaddyfile(model, { includeGlobal: false });

          if (input.kind === 'enabled') {
            const hc = input.healthCheck as HealthCheck;
            expect(hasEmittableHealthCheck(hc)).toBe(true);
            expect(caddyfile).toContain(`health_uri ${hc.path.trim()}`);
            if (hc.interval?.trim()) {
              expect(caddyfile).toContain(`health_interval ${hc.interval.trim()}`);
            } else {
              expect(caddyfile).not.toMatch(/\bhealth_interval\b/);
            }
            if (hc.timeout?.trim()) {
              expect(caddyfile).toContain(`health_timeout ${hc.timeout.trim()}`);
            } else {
              expect(caddyfile).not.toMatch(/\bhealth_timeout\b/);
            }
          } else {
            // Req 3.2：关闭或全空时省略全部 health 指令，不写空块
            expect(caddyfile).not.toMatch(/\bhealth_uri\b/);
            expect(caddyfile).not.toMatch(/\bhealth_interval\b/);
            expect(caddyfile).not.toMatch(/\bhealth_timeout\b/);
          }

          if (input.lbPolicy) {
            expect(caddyfile).toContain(`lb_policy ${input.lbPolicy}`);
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('unit: 启用 health + lb 时 reverse_proxy 为块并含指令', () => {
    const model = modelWithReverseProxy({
      healthCheck: { path: '/healthz', interval: '10s', timeout: '2s' },
      lbPolicy: 'least_conn'
    });
    const out = buildCaddyfile(model, { includeGlobal: false });
    expect(out).toContain('reverse_proxy 127.0.0.1:8080 {');
    expect(out).toContain('lb_policy least_conn');
    expect(out).toContain('health_uri /healthz');
    expect(out).toContain('health_interval 10s');
    expect(out).toContain('health_timeout 2s');
  });

  it('unit: 无 health 无 lb 无 transport 时不写 reverse_proxy 空块', () => {
    const model = modelWithReverseProxy({});
    const out = buildCaddyfile(model, { includeGlobal: false });
    expect(out).toContain('reverse_proxy 127.0.0.1:8080');
    expect(out).not.toContain('reverse_proxy 127.0.0.1:8080 {');
    expect(out).not.toMatch(/\bhealth_uri\b/);
  });

  it('unit: 解析 reverse_proxy 块中的 health 与 lb_policy', () => {
    const raw = `
app.example.com {
  handle {
    reverse_proxy 10.0.0.1:9000 {
      lb_policy ip_hash
      health_uri /ready
      health_interval 5s
      health_timeout 1s
    }
  }
}
`.trim();
    const model = parseCaddyfileToModules(raw);
    const handle = model.sites[0]?.routes[0]?.handles.find(h => h.type === 'reverse_proxy');
    expect(handle?.upstream).toContain('10.0.0.1:9000');
    expect(handle?.lbPolicy).toBe('ip_hash');
    expect(handle?.healthCheck).toEqual({
      path: '/ready',
      interval: '5s',
      timeout: '1s'
    });
  });

  it('unit: path 不以 / 开头返回中文字段错误', () => {
    const errs = validateHealthCheckFields({ path: 'healthz', interval: '10s' });
    expect(errs.some(e => e.includes('路径') && e.includes('/'))).toBe(true);
  });

  it('unit: interval/timeout 非法格式返回中文错误', () => {
    const errs = validateHealthCheckFields({ path: '/ok', interval: '10', timeout: 'fast' });
    expect(errs.some(e => e.includes('间隔'))).toBe(true);
    expect(errs.some(e => e.includes('超时'))).toBe(true);
  });

  it('unit: validateStructuredConfig 拒绝非法 health path', () => {
    const model = modelWithReverseProxy({
      healthCheck: { path: 'no-slash', interval: '1s' }
    });
    const errors = validateStructuredConfig(model);
    expect(errors.some(e => e.includes('健康检查') && e.includes('/'))).toBe(true);
  });
});
