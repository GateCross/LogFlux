/**
 * 一步建站的纯函数层：表单校验、站点块渲染、追加进现有 Caddyfile。
 *
 * 建站走 raw 追加而不是结构化重渲染：新块拼在文末，其余字节一概不动，
 * 与「分块工作台」的编辑路径互不影响。
 */

/** manual = 已有证书：从证书库下拉选择，按 tls 双路径写入站点块 */
export type CreateTlsMode = 'auto' | 'dns' | 'internal' | 'manual' | 'off';

export type CreateLbPolicy = 'round_robin' | 'least_conn' | 'ip_hash';

export interface CreateSiteForm {
  domains: string[];
  upstream: string;
  tlsMode: CreateTlsMode;
  /** manual 模式的证书/私钥路径（Caddy 容器内路径，由证书库下拉写入） */
  manualCertPath: string;
  manualKeyPath: string;
  lbPolicy: CreateLbPolicy;
  healthEnabled: boolean;
  healthPath: string;
  healthInterval: string;
  healthTimeout: string;
}

export function createDefaultSiteForm(): CreateSiteForm {
  return {
    domains: [],
    upstream: 'localhost:8080',
    tlsMode: 'auto',
    manualCertPath: '',
    manualKeyPath: '',
    lbPolicy: 'round_robin',
    healthEnabled: false,
    healthPath: '/health',
    healthInterval: '10s',
    healthTimeout: '5s',
  };
}

const portOnlyRe = /^:\d+$/;
const localhostHostRe = /^localhost(?::\d+)?$/i;
const domainRe = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$/;
const ipv4Re = /^\d{1,3}(\.\d{1,3}){3}$/;
const durationRe = /^[1-9]\d*(ms|s|m|h)$/;

export function isValidDomainOrListen(value: string): boolean {
  const token = value.trim();
  if (!token) return false;
  if (portOnlyRe.test(token)) return true;
  if (localhostHostRe.test(token)) return true;
  if (domainRe.test(token) || ipv4Re.test(token)) return true;
  const cut = token.lastIndexOf(':');
  if (cut > 0) {
    const host = token.slice(0, cut);
    const port = token.slice(cut + 1);
    if (/^\d+$/.test(port)) {
      return domainRe.test(host) || ipv4Re.test(host) || host.toLowerCase() === 'localhost';
    }
  }
  return false;
}

/** 已存在于配置里的域名（块头任意位置出现即算，防重复建站）。 */
export function findDuplicateDomains(config: string, domains: string[]): string[] {
  return domains.filter((domain) =>
    new RegExp(`(^|\\s)${domain.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(\\s|\\{|$)`, 'm').test(config),
  );
}

/**
 * 校验建站表单；返回中文错误列表（空 = 通过）。
 * dns 模式依赖配置里已有的 (cloudflare) snippet，缺失时直接报错而不是生成坏配置。
 */
export function validateCreateSiteForm(form: CreateSiteForm, currentConfig: string): string[] {
  const errors: string[] = [];
  const domains = form.domains.map((item) => item.trim()).filter(Boolean);
  if (domains.length === 0) {
    errors.push('至少填写一个域名或监听地址');
  }
  const invalid = domains.filter((item) => !isValidDomainOrListen(item));
  if (invalid.length > 0) {
    errors.push(`域名格式不合法: ${invalid.join(', ')}`);
  }
  const duplicated = findDuplicateDomains(currentConfig, domains);
  if (duplicated.length > 0) {
    errors.push(`配置里已存在这些域名: ${duplicated.join(', ')}`);
  }
  if (!form.upstream.trim()) {
    errors.push('上游地址不能为空');
  }
  if (form.tlsMode === 'dns' && !/^\(cloudflare\)\s*\{/m.test(currentConfig)) {
    errors.push('DNS 签发需要配置里已定义 (cloudflare) 片段');
  }
  // 已有证书模式只认证书库下拉写入的双路径，不提供手填入口
  if (
    form.tlsMode === 'manual' &&
    (!form.manualCertPath.trim() || !form.manualKeyPath.trim())
  ) {
    errors.push('已有证书模式需先从证书库选择一份证书');
  }
  if (form.healthEnabled) {
    if (form.healthPath.trim() && !form.healthPath.trim().startsWith('/')) {
      errors.push('健康检查路径必须以 / 开头');
    }
    for (const [label, value] of [['间隔', form.healthInterval], ['超时', form.healthTimeout]] as const) {
      if (value.trim() && !durationRe.test(value.trim())) {
        errors.push(`健康检查${label}格式无效，应为正整数加单位（ms|s|m|h）`);
      }
    }
  }
  return errors;
}

/** 渲染新站点块（两空格缩进，LF）。 */
export function buildCreatedSiteBlock(form: CreateSiteForm): string {
  const domains = form.domains.map((item) => item.trim()).filter(Boolean);
  const lines: string[] = [`${domains.join(' ')} {`];
  if (form.tlsMode === 'dns') lines.push('  import cloudflare');
  if (form.tlsMode === 'off') lines.push('  tls off');
  if (form.tlsMode === 'internal') lines.push('  tls internal');
  // 已有证书：按 manual 形式写 tls 双路径；路径残缺时不渲染（防半行指令），由校验拦截
  const manualCert = form.manualCertPath.trim();
  const manualKey = form.manualKeyPath.trim();
  if (form.tlsMode === 'manual' && manualCert && manualKey) {
    lines.push(`  tls ${manualCert} ${manualKey}`);
  }

  const upstream = form.upstream.trim();
  const healthPath = form.healthEnabled ? form.healthPath.trim() : '';
  const healthPathValid = healthPath.startsWith('/');
  const needBlock =
    (form.lbPolicy !== 'round_robin' && Boolean(form.lbPolicy)) ||
    (form.healthEnabled && healthPathValid);

  if (!needBlock) {
    lines.push('  handle {', `    reverse_proxy ${upstream}`, '  }');
  } else {
    lines.push('  handle {', `    reverse_proxy ${upstream} {`);
    if (form.lbPolicy && form.lbPolicy !== 'round_robin') {
      lines.push(`      lb_policy ${form.lbPolicy}`);
    }
    if (form.healthEnabled && healthPathValid) {
      lines.push(`      health_uri ${healthPath}`);
      if (form.healthInterval.trim()) lines.push(`      health_interval ${form.healthInterval.trim()}`);
      if (form.healthTimeout.trim()) lines.push(`      health_timeout ${form.healthTimeout.trim()}`);
    }
    lines.push('    }', '  }');
  }
  lines.push('}');
  return lines.join('\n');
}

/** 新块追加到配置末尾：中间恰好一个空行，收尾恰好一个换行。 */
export function appendSiteBlockToConfig(config: string, block: string): string {
  const base = config.replace(/\s+$/, '');
  if (!base) return `${block}\n`;
  return `${base}\n\n${block}\n`;
}

/** 新站点的块头地址（import waf_protect 挂载时按此对齐 availableSites）。 */
export function createdSiteAddress(form: CreateSiteForm): string {
  return form.domains.map((item) => item.trim()).filter(Boolean).join(' ');
}
