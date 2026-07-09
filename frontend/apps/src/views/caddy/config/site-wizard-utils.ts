import type { CaddyFormModel, HealthCheck } from './types';
import {
  buildSiteFromQuickDraft,
  createQuickSiteDraft,
  mergeQuickConfigDrafts,
  type QuickLbPolicy,
  type QuickSiteDraft,
  type QuickTlsMode,
} from './quick-config-utils';
import { siteDomainRe, siteIpv4Re } from './caddy-config-utils';
import { validateHealthCheckFields, buildCaddyfile } from './caddy-config-parser';
import { formatCaddyfile } from './caddy-config-diff';

/** 站点创建向导步骤：域名 → 上游 → TLS → 可选 WAF → Preview → Apply_Path */
export type SiteWizardStep =
  | 'domain'
  | 'upstream'
  | 'tls'
  | 'waf'
  | 'preview'
  | 'apply';

export type SiteWizardWafMode = 'detectiononly' | 'off' | 'on';
export type SiteWizardWafStrength = 'balanced' | 'high_blocking' | 'low_fp';
export type SiteWizardWafAudit = 'off' | 'on' | 'relevantonly';

/** 可选 WAF 意图（草稿阶段不调用 WAF /load） */
export interface SiteWizardWafDraft {
  /** 是否在应用站点后携带 WAF 意图（仍须用户确认 Apply_Path） */
  enabled: boolean;
  mode: SiteWizardWafMode;
  strength: SiteWizardWafStrength;
  audit: SiteWizardWafAudit;
}

export interface SiteWizardState {
  name: string;
  domains: string[];
  upstream: string;
  lbPolicy: QuickLbPolicy;
  healthEnabled: boolean;
  healthPath: string;
  healthInterval: string;
  healthTimeout: string;
  tlsMode: QuickTlsMode;
  waf: SiteWizardWafDraft;
}

export interface SiteWizardStepMeta {
  key: SiteWizardStep;
  title: string;
  description: string;
}

export const SITE_WIZARD_STEPS: SiteWizardStepMeta[] = [
  { key: 'domain', title: '域名', description: '站点名称与域名 / 监听地址' },
  { key: 'upstream', title: '上游', description: '反代目标、负载策略与可选健康检查' },
  { key: 'tls', title: 'TLS', description: '站点 TLS 模式' },
  { key: 'waf', title: '可选 WAF', description: '可选防火墙意图，默认不热加载' },
  { key: 'preview', title: '预览', description: '生成草稿与 Caddyfile 预览（仅 /adapt）' },
  { key: 'apply', title: '应用', description: '确认后走既有 Apply_Path；默认可仅写入草稿' },
];

const portOnlyRe = /^:\d+$/;
const localhostHostRe = /^localhost(?::\d+)?$/i;

export function createSiteWizardState(partial?: Partial<SiteWizardState>): SiteWizardState {
  return {
    name: partial?.name ?? '',
    domains: partial?.domains ? [...partial.domains] : [],
    upstream: partial?.upstream ?? 'localhost:8080',
    lbPolicy: partial?.lbPolicy ?? 'round_robin',
    healthEnabled: partial?.healthEnabled ?? false,
    healthPath: partial?.healthPath ?? '/health',
    healthInterval: partial?.healthInterval ?? '10s',
    healthTimeout: partial?.healthTimeout ?? '5s',
    tlsMode: partial?.tlsMode ?? 'auto',
    waf: {
      enabled: partial?.waf?.enabled ?? false,
      mode: partial?.waf?.mode ?? 'detectiononly',
      strength: partial?.waf?.strength ?? 'balanced',
      audit: partial?.waf?.audit ?? 'relevantonly',
    },
  };
}

function isValidDomainOrListen(value: string): boolean {
  const token = value.trim();
  if (!token) return false;
  if (portOnlyRe.test(token)) return true;
  if (localhostHostRe.test(token)) return true;
  if (siteDomainRe.test(token) || siteIpv4Re.test(token)) return true;
  // host:port 形式
  const [host = '', port] = token.split(':');
  if (port && /^\d+$/.test(port)) {
    return siteDomainRe.test(host) || siteIpv4Re.test(host) || host.toLowerCase() === 'localhost';
  }
  return false;
}

function buildHealthCheck(state: SiteWizardState): HealthCheck | undefined {
  if (!state.healthEnabled) return undefined;
  return {
    path: state.healthPath.trim(),
    interval: state.healthInterval.trim() || undefined,
    timeout: state.healthTimeout.trim() || undefined,
  };
}

/**
 * 校验向导某一步；返回中文错误列表（空 = 通过）。
 */
export function validateWizardStep(state: SiteWizardState, step: SiteWizardStep): string[] {
  const errors: string[] = [];

  if (step === 'domain') {
    if (!state.name.trim()) {
      errors.push('站点名称不能为空');
    }
    const domains = state.domains.map((d) => d.trim()).filter(Boolean);
    if (domains.length === 0) {
      errors.push('至少配置一个域名或监听地址');
    }
    const invalid = domains.filter((d) => !isValidDomainOrListen(d));
    if (invalid.length) {
      errors.push(`域名格式不合法: ${invalid.join(', ')}`);
    }
  }

  if (step === 'upstream') {
    if (!state.upstream.trim()) {
      errors.push('上游地址或池名不能为空');
    }
    if (state.healthEnabled) {
      errors.push(
        ...validateHealthCheckFields(buildHealthCheck(state), '站点级健康检查'),
      );
    }
  }

  if (step === 'tls') {
    if (!['auto', 'off', 'internal'].includes(state.tlsMode)) {
      errors.push('TLS 模式无效');
    }
  }

  if (step === 'waf') {
    if (state.waf.enabled && state.waf.mode === 'off') {
      // 开启意图但模式为 off 无意义，给出提示性校验
      errors.push('已启用 WAF 意图时，引擎模式不能为「关闭」');
    }
  }

  return errors;
}

/**
 * 校验从起点到当前步骤（含）的全部必填项，用于 Preview / Apply 前。
 */
export function validateWizardUpTo(state: SiteWizardState, step: SiteWizardStep): string[] {
  const order = SITE_WIZARD_STEPS.map((item) => item.key);
  const end = order.indexOf(step);
  if (end < 0) return ['未知向导步骤'];
  const errors: string[] = [];
  for (let i = 0; i <= end; i++) {
    const key = order[i]!;
    // preview/apply 依赖 domain/upstream/tls；waf 仅在启用时校验
    if (key === 'preview' || key === 'apply') continue;
    if (key === 'waf' && !state.waf.enabled) continue;
    errors.push(...validateWizardStep(state, key));
  }
  return errors;
}

/**
 * 将向导状态转为 QuickSiteDraft（纯草稿，不触发任何 API /load）。
 */
export function buildQuickSiteDraftFromWizard(
  state: SiteWizardState,
  id?: string,
): QuickSiteDraft {
  const domains = state.domains.map((d) => d.trim()).filter(Boolean);
  const name = state.name.trim() || domains[0] || '新站点';
  return createQuickSiteDraft({
    id,
    name,
    enabled: true,
    domains,
    tlsMode: state.tlsMode,
    mode: 'reverse_proxy',
    upstream: state.upstream.trim(),
    lbPolicy: state.lbPolicy,
    healthCheck: buildHealthCheck(state),
  });
}

/**
 * 基于向导草稿生成本地 Caddyfile 预览（纯函数，不调用 /adapt 或 /load）。
 * baseModel 可选：传入当前工作台 model 时做合并预览。
 */
export function buildWizardLocalCaddyfilePreview(
  state: SiteWizardState,
  baseModel?: CaddyFormModel,
  draftId?: string,
): string {
  const draft = buildQuickSiteDraftFromWizard(state, draftId);
  if (baseModel) {
    const merged = mergeQuickConfigDrafts(baseModel, [draft]);
    return formatCaddyfile(buildCaddyfile(merged));
  }
  const site = buildSiteFromQuickDraft(draft);
  return formatCaddyfile(
    buildCaddyfile({
      schemaVersion: 1,
      global: { raw: '' },
      upstreams: [],
      sites: [site],
    }),
  );
}

/**
 * 向导步骤索引（0-based）。
 */
export function wizardStepIndex(step: SiteWizardStep): number {
  return SITE_WIZARD_STEPS.findIndex((item) => item.key === step);
}
export function wizardStepAt(index: number): SiteWizardStep {
  const clamped = Math.max(0, Math.min(SITE_WIZARD_STEPS.length - 1, index));
  return SITE_WIZARD_STEPS[clamped]!.key;
}

/**
 * 是否允许「仅写入草稿」：始终 true（默认路径，禁止自动 /load）。
 * 导出为显式语义，便于测试与 UI 绑定。
 */
export function wizardAllowsDraftOnly(): true {
  return true;
}

/**
 * 是否允许自动 /load：恒为 false（硬约束：无确认不热加载）。
 */
export function wizardAutoLoadEnabled(): false {
  return false;
}
