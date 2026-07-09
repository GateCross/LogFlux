import { describe, expect, it } from 'vitest';

import { buildSiteFromQuickDraft } from './quick-config-utils';
import {
  buildQuickSiteDraftFromWizard,
  buildWizardLocalCaddyfilePreview,
  createSiteWizardState,
  SITE_WIZARD_STEPS,
  validateWizardStep,
  validateWizardUpTo,
  wizardAllowsDraftOnly,
  wizardAutoLoadEnabled,
  wizardStepAt,
  wizardStepIndex,
} from './site-wizard-utils';

describe('site-wizard-utils', () => {
  it('步骤顺序为 域名 → 上游 → TLS → 可选 WAF → Preview → Apply', () => {
    expect(SITE_WIZARD_STEPS.map((s) => s.key)).toEqual([
      'domain',
      'upstream',
      'tls',
      'waf',
      'preview',
      'apply',
    ]);
    expect(wizardStepIndex('domain')).toBe(0);
    expect(wizardStepIndex('apply')).toBe(5);
    expect(wizardStepAt(3)).toBe('waf');
  });

  it('默认不自动 /load，允许仅产出草稿', () => {
    expect(wizardAutoLoadEnabled()).toBe(false);
    expect(wizardAllowsDraftOnly()).toBe(true);
  });

  it('域名步骤：空名称 / 空域名 / 非法域名报中文错误', () => {
    expect(validateWizardStep(createSiteWizardState({ name: '', domains: [] }), 'domain')).toEqual(
      expect.arrayContaining(['站点名称不能为空', '至少配置一个域名或监听地址']),
    );
    expect(
      validateWizardStep(
        createSiteWizardState({ name: 'demo', domains: ['not a domain!!'] }),
        'domain',
      )[0],
    ).toMatch(/域名格式不合法/);
    expect(
      validateWizardStep(
        createSiteWizardState({ name: 'demo', domains: ['example.com', ':8080'] }),
        'domain',
      ),
    ).toEqual([]);
  });

  it('上游步骤：空上游与非法 health path 报错', () => {
    expect(
      validateWizardStep(createSiteWizardState({ upstream: '  ' }), 'upstream')[0],
    ).toMatch(/上游/);
    const errs = validateWizardStep(
      createSiteWizardState({
        upstream: '127.0.0.1:8080',
        healthEnabled: true,
        healthPath: 'health',
      }),
      'upstream',
    );
    expect(errs.some((e) => e.includes('/') || e.includes('path') || e.includes('路径'))).toBe(
      true,
    );
  });

  it('未启用 WAF 时跳过 WAF 步骤校验', () => {
    const state = createSiteWizardState({
      name: 'ok',
      domains: ['ok.example.com'],
      upstream: '127.0.0.1:9000',
      waf: { enabled: false, mode: 'off', strength: 'balanced', audit: 'relevantonly' },
    });
    // mode=off 仅在 enabled 时才报错
    expect(validateWizardUpTo(state, 'preview')).toEqual([]);
  });

  it('启用 WAF 且模式为 off 时报中文错误', () => {
    const errs = validateWizardStep(
      createSiteWizardState({
        waf: { enabled: true, mode: 'off', strength: 'balanced', audit: 'relevantonly' },
      }),
      'waf',
    );
    expect(errs.some((e) => e.includes('WAF'))).toBe(true);
  });

  it('buildQuickSiteDraftFromWizard 产出可 round-trip 的 QuickSiteDraft', () => {
    const state = createSiteWizardState({
      name: '官网',
      domains: ['www.example.com'],
      upstream: 'app-pool',
      lbPolicy: 'least_conn',
      healthEnabled: true,
      healthPath: '/ready',
      healthInterval: '15s',
      healthTimeout: '3s',
      tlsMode: 'internal',
      waf: { enabled: true, mode: 'on', strength: 'high_blocking', audit: 'relevantonly' },
    });

    const draft = buildQuickSiteDraftFromWizard(state, 'wiz-1');
    expect(draft.id).toBe('wiz-1');
    expect(draft.name).toBe('官网');
    expect(draft.domains).toEqual(['www.example.com']);
    expect(draft.upstream).toBe('app-pool');
    expect(draft.lbPolicy).toBe('least_conn');
    expect(draft.tlsMode).toBe('internal');
    expect(draft.mode).toBe('reverse_proxy');
    expect(draft.healthCheck).toEqual({
      path: '/ready',
      interval: '15s',
      timeout: '3s',
    });

    // 草稿可进入既有 config model
    const site = buildSiteFromQuickDraft(draft);
    expect(site.domains).toEqual(['www.example.com']);
    expect(site.tls?.mode).toBe('internal');
    expect(site.routes[0]?.handles[0]?.upstream).toBe('app-pool');
    expect(site.routes[0]?.handles[0]?.lbPolicy).toBe('least_conn');
    expect(site.routes[0]?.handles[0]?.healthCheck?.path).toBe('/ready');
  });

  it('关闭健康检查时 draft 不含 healthCheck', () => {
    const draft = buildQuickSiteDraftFromWizard(
      createSiteWizardState({
        name: 's',
        domains: ['a.example.com'],
        healthEnabled: false,
        healthPath: '/health',
      }),
    );
    expect(draft.healthCheck).toBeUndefined();
  });

  it('validateWizardUpTo 在 preview 前要求域名与上游', () => {
    const incomplete = createSiteWizardState({ name: '', domains: [], upstream: '' });
    const errors = validateWizardUpTo(incomplete, 'preview');
    expect(errors.length).toBeGreaterThan(0);

    const complete = createSiteWizardState({
      name: 'ok',
      domains: ['ok.example.com'],
      upstream: '127.0.0.1:9000',
    });
    expect(validateWizardUpTo(complete, 'preview')).toEqual([]);
  });

  it('buildWizardLocalCaddyfilePreview 含域名与上游且不触发副作用', () => {
    const text = buildWizardLocalCaddyfilePreview(
      createSiteWizardState({
        name: 'preview-site',
        domains: ['preview.example.com'],
        upstream: '10.0.0.1:8080',
        lbPolicy: 'ip_hash',
        healthEnabled: true,
        healthPath: '/healthz',
        healthInterval: '10s',
        tlsMode: 'auto',
      }),
    );
    expect(text).toContain('preview.example.com');
    expect(text).toContain('reverse_proxy');
    expect(text).toContain('10.0.0.1:8080');
    expect(text).toMatch(/health_uri\s+\/healthz/);
    expect(text).toMatch(/lb_policy\s+ip_hash/);
  });
});
