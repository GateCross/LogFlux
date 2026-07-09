import * as fc from 'fast-check';
import { describe, expect, it } from 'vitest';

import { isEditableSite } from './caddy-config-blocks';
import {
  analyzeSiteForQuickConfig,
  buildSiteFromQuickDraft,
  createQuickSiteDraft,
  type QuickLbPolicy
} from './quick-config-utils';
import type { Handle, HealthCheck, Site } from './types';

const SIMPLE_LB_POLICIES: QuickLbPolicy[] = ['round_robin', 'least_conn', 'ip_hash'];

const durationArb = fc.tuple(fc.integer({ min: 1, max: 3600 }), fc.constantFrom('ms', 's', 'm', 'h')).map(
  ([n, unit]) => `${n}${unit}`
);

const healthPathArb = fc
  .string({ minLength: 1, maxLength: 24 })
  .filter(s => /^[a-zA-Z0-9_-]+$/.test(s))
  .map(s => `/${s}`);

const healthCheckArb: fc.Arbitrary<HealthCheck> = fc.record({
  path: healthPathArb,
  interval: fc.option(durationArb, { nil: undefined }),
  timeout: fc.option(durationArb, { nil: undefined })
});

const lbPolicyArb = fc.constantFrom<QuickLbPolicy>(...SIMPLE_LB_POLICIES);

const domainArb = fc
  .tuple(
    fc.string({ minLength: 1, maxLength: 12 }).filter(s => /^[a-z0-9]+$/.test(s)),
    fc.constantFrom('example.com', 'local.test', 'svc.internal')
  )
  .map(([sub, root]) => `${sub}.${root}`);

const poolNameArb = fc
  .string({ minLength: 1, maxLength: 16 })
  .filter(s => /^[a-zA-Z][a-zA-Z0-9_-]*$/.test(s));

const directUpstreamArb = fc
  .tuple(fc.ipV4(), fc.integer({ min: 1, max: 65535 }))
  .map(([ip, port]) => `${ip}:${port}`);

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

function baseSimpleReverseProxySite(opts: {
  id: string;
  name: string;
  domains: string[];
  upstream: string;
  lbPolicy?: QuickLbPolicy;
  healthCheck?: HealthCheck;
}): Site {
  const handle: Handle = {
    id: `${opts.id}-handle`,
    type: 'reverse_proxy',
    enabled: true,
    upstream: opts.upstream,
    lbPolicy: opts.lbPolicy,
    healthCheck: opts.healthCheck,
    tlsInsecureSkipVerify: false,
    transportProtocol: ''
  };

  return {
    id: opts.id,
    name: opts.name,
    enabled: true,
    domains: opts.domains,
    tls: { mode: 'auto' },
    imports: [],
    geoip2Vars: [],
    encode: [],
    routes: [
      {
        id: `${opts.id}-route`,
        name: '默认路由',
        enabled: true,
        match: emptyMatch(),
        handles: [handle],
        logAppend: []
      }
    ]
  };
}

describe('quick-config-utils simple classification & draft round-trip', () => {
  /**
   * Property 1: Simple 分类放宽
   * For any reverse_proxy that only additionally has Health_Check, defined pool refs,
   * or common Lb_Policy, classification result is Simple_Site.
   * **Validates: Requirements 2.1**
   */
  it('Property 1: Simple 分类放宽', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.uuid(),
          name: fc.string({ minLength: 1, maxLength: 20 }).filter(s => s.trim().length > 0),
          domains: fc.array(domainArb, { minLength: 1, maxLength: 3 }),
          usePool: fc.boolean(),
          poolName: poolNameArb,
          directUpstream: directUpstreamArb,
          lbPolicy: fc.option(lbPolicyArb, { nil: undefined }),
          healthCheck: fc.option(healthCheckArb, { nil: undefined })
        }),
        input => {
          const upstream = input.usePool ? input.poolName : input.directUpstream;
          const upstreamNames = new Set(input.usePool ? [input.poolName] : []);
          const site = baseSimpleReverseProxySite({
            id: input.id,
            name: input.name,
            domains: input.domains,
            upstream,
            lbPolicy: input.lbPolicy,
            healthCheck: input.healthCheck
          });

          const analyzed = analyzeSiteForQuickConfig(site, upstreamNames);
          expect(analyzed.kind).toBe('simple');
          // isEditableSite 与分类同源
          expect(isEditableSite(site, upstreamNames)).toBe(true);

          if (analyzed.kind === 'simple') {
            expect(analyzed.draft.mode).toBe('reverse_proxy');
            expect(analyzed.draft.upstream).toBe(upstream);
            if (input.lbPolicy) {
              expect(analyzed.draft.lbPolicy).toBe(input.lbPolicy);
            }
            if (input.healthCheck) {
              expect(analyzed.draft.healthCheck).toEqual(input.healthCheck);
            }
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property 2: QuickDraft round-trip 保留 health/lb
   * For any Simple reverse_proxy with healthCheck and common lbPolicy,
   * after draft → buildSiteFromQuickDraft → re-analyze,
   * path / interval / timeout / lbPolicy / upstream semantics are preserved.
   * **Validates: Requirements 2.2, 2.4**
   */
  it('Property 2: QuickDraft round-trip 保留 health/lb', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.uuid(),
          name: fc.string({ minLength: 1, maxLength: 20 }).filter(s => s.trim().length > 0),
          domains: fc.array(domainArb, { minLength: 1, maxLength: 3 }),
          usePool: fc.boolean(),
          poolName: poolNameArb,
          directUpstream: directUpstreamArb,
          lbPolicy: lbPolicyArb,
          healthCheck: healthCheckArb
        }),
        input => {
          const upstream = input.usePool ? input.poolName : input.directUpstream;
          const upstreamNames = new Set(input.usePool ? [input.poolName] : []);
          const original = baseSimpleReverseProxySite({
            id: input.id,
            name: input.name,
            domains: input.domains,
            upstream,
            lbPolicy: input.lbPolicy,
            healthCheck: input.healthCheck
          });

          const first = analyzeSiteForQuickConfig(original, upstreamNames);
          expect(first.kind).toBe('simple');
          if (first.kind !== 'simple') return;

          // 草稿字段应暴露 health / lb（Req 2.2）
          expect(first.draft.lbPolicy).toBe(input.lbPolicy);
          expect(first.draft.healthCheck).toEqual(input.healthCheck);
          expect(first.draft.upstream).toBe(upstream);

          const rebuilt = buildSiteFromQuickDraft(first.draft);
          const rebuiltHandle = rebuilt.routes[0]?.handles[0];
          expect(rebuiltHandle?.type).toBe('reverse_proxy');
          expect(rebuiltHandle?.upstream).toBe(upstream.trim());
          expect(rebuiltHandle?.lbPolicy).toBe(input.lbPolicy);
          expect(rebuiltHandle?.healthCheck).toEqual(input.healthCheck);

          const second = analyzeSiteForQuickConfig(rebuilt, upstreamNames);
          expect(second.kind).toBe('simple');
          if (second.kind !== 'simple') return;

          expect(second.draft.upstream).toBe(upstream.trim());
          expect(second.draft.lbPolicy).toBe(input.lbPolicy);
          expect(second.draft.healthCheck).toEqual(input.healthCheck);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('unit: createQuickSiteDraft 可写入 healthCheck 与 lbPolicy', () => {
    const draft = createQuickSiteDraft({
      mode: 'reverse_proxy',
      upstream: 'pool-a',
      lbPolicy: 'least_conn',
      healthCheck: { path: '/healthz', interval: '10s', timeout: '2s' }
    });
    expect(draft.lbPolicy).toBe('least_conn');
    expect(draft.healthCheck).toEqual({ path: '/healthz', interval: '10s', timeout: '2s' });
  });

  it('unit: 复杂特征仍判 complex 并给出中文原因', () => {
    const site = baseSimpleReverseProxySite({
      id: 'c1',
      name: 'complex-site',
      domains: ['a.example.com'],
      upstream: '127.0.0.1:8080',
      lbPolicy: 'round_robin',
      healthCheck: { path: '/ok' }
    });
    site.imports = ['snippet'];
    site.routes[0]!.handles[0]!.transportProtocol = 'http';

    const analyzed = analyzeSiteForQuickConfig(site, new Set());
    expect(analyzed.kind).toBe('complex');
    if (analyzed.kind === 'complex') {
      expect(analyzed.summary.reasons.some(r => r.includes('import'))).toBe(true);
      expect(analyzed.summary.reasons.some(r => r.includes('传输协议'))).toBe(true);
    }
  });
});
