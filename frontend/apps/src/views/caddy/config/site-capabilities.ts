import type { SiteLogSettings } from './site-log-settings';
import { detectChildIndent, findHeaderLine } from './caddy-config-utils';
import {
  parseSiteLogImport,
  renderSiteImportBlock,
  SITE_LOG_IMPORT_HEAD_RE,
} from './site-log-settings';

/**
 * 站点能力识别与写回（纯函数，无 API 依赖）。
 *
 * 站点块文本是唯一事实来源：这里只做"从块文本识别能力"与"把能力行写回块文本"，
 * 不引入第二个真相源。识别结果三态：on / off / unknown，unknown 表示文本里有
 * 相关内容但认不出来，UI 只能禁用开关并引导用户去原文模式。
 */

export type SiteCapabilityStatus = 'on' | 'off' | 'unknown';

export interface SiteCapabilitySnapshot {
  status: SiteCapabilityStatus;
  /** on/unknown 时给用户看的补充说明 */
  note?: string;
}

export interface SiteTlsSnapshot {
  mode: 'auto' | 'dns' | 'internal' | 'manual' | 'off';
  status: SiteCapabilityStatus;
  note?: string;
}

export interface SiteCapabilitiesSnapshot {
  waf: SiteCapabilitySnapshot;
  geo: SiteCapabilitySnapshot;
  compression: SiteCapabilitySnapshot;
  securityHeaders: SiteCapabilitySnapshot;
  accessLog: SiteCapabilitySnapshot;
  errorPage: SiteCapabilitySnapshot;
  tls: SiteTlsSnapshot;
  /** 未归类到任何能力的站点级 import（health_check / api_proxy 等业务片段） */
  otherImports: string[];
}

export type SiteCapabilityKey =
  | 'waf'
  | 'geo'
  | 'compression'
  | 'securityHeaders'
  | 'accessLog'
  | 'errorPage';

interface CapabilityAnchor {
  importName: string;
  /** 内联写法与 snippet 等价，识别为开启 */
  inlineOn?: RegExp[];
  /** 内联痕迹无法确认属于本能力，判 unknown */
  inlineUnknown?: RegExp[];
}

/** 每个能力对应的 import snippet 名与内联痕迹关键词 */
const CAPABILITY_ANCHORS: Record<SiteCapabilityKey, CapabilityAnchor> = {
  waf: { importName: 'waf_protect', inlineUnknown: [/coraza_waf/] },
  geo: {
    importName: 'geo_auth',
    inlineOn: [/forward_auth\s+localhost:8888/],
  },
  compression: { importName: 'compression', inlineOn: [/^[ \t]*encode\s/m] },
  securityHeaders: { importName: 'security_headers', inlineUnknown: [/^[ \t]*header\s/m] },
  accessLog: { importName: 'subdomain-log', inlineOn: [/^[ \t]*log\s*\{/m] },
  errorPage: { importName: 'deny_403_redirect', inlineUnknown: [/^[ \t]*handle_errors/m] },
};

/** 已归类的 snippet 名；import 行按此分派到能力（quick-config 分类器也用它排除误判） */
export const CLASSIFIED_IMPORTS = new Set([
  ...Object.values(CAPABILITY_ANCHORS).map(item => item.importName),
  'cloudflare',
]);

function importLineRegex(name: string): RegExp {
  // import <name> 或 import <name> <args...>；多行模式匹配整块，单行探测同样适用
  return new RegExp(`^\\s*import\\s+${name}(?:\\s|$)`, 'm');
}

function detectImport(block: string, name: string): boolean {
  return importLineRegex(name).test(`${block}\n`);
}

function detectCapability(block: string, key: SiteCapabilityKey): SiteCapabilitySnapshot {
  const anchor = CAPABILITY_ANCHORS[key];
  if (detectImport(block, anchor.importName)) {
    return { status: 'on' };
  }
  for (const pattern of anchor.inlineOn ?? []) {
    if (pattern.test(block)) {
      return { status: 'on', note: '内联配置' };
    }
  }
  for (const pattern of anchor.inlineUnknown ?? []) {
    if (pattern.test(block)) {
      return { status: 'unknown', note: '检测到内联配置，无法结构化识别，请使用原文模式' };
    }
  }
  return { status: 'off' };
}

function detectTls(block: string): SiteTlsSnapshot {
  if (detectImport(block, 'cloudflare')) {
    return { mode: 'dns', status: 'on' };
  }
  const inline = block.match(/^[ \t]*tls\s+(.+)$/m);
  if (inline) {
    const rest = (inline[1] ?? '').trim();
    if (rest === 'off') return { mode: 'off', status: 'on' };
    if (rest === 'internal') return { mode: 'internal', status: 'on' };
    if (rest.includes(' ')) return { mode: 'manual', status: 'on' };
    return { mode: 'auto', status: 'unknown', note: `检测到无法识别的 tls 配置：${rest}` };
  }
  if (/^[ \t]*tls\s*\{/m.test(block)) {
    return { mode: 'auto', status: 'unknown', note: '检测到内联 tls 块，请使用原文模式' };
  }
  return { mode: 'auto', status: 'off' };
}

function collectOtherImports(block: string): string[] {
  const result: string[] = [];
  for (const line of block.split('\n')) {
    const match = line.match(/^\s*import\s+([^\s{]+)/);
    const name = match?.[1];
    if (name && !CLASSIFIED_IMPORTS.has(name) && !result.includes(name)) {
      result.push(name);
    }
  }
  return result;
}

/** 从站点块文本识别全部能力状态。 */
export function extractSiteCapabilities(siteBlock: string): SiteCapabilitiesSnapshot {
  const block = siteBlock ?? '';
  return {
    waf: detectCapability(block, 'waf'),
    geo: detectCapability(block, 'geo'),
    compression: detectCapability(block, 'compression'),
    securityHeaders: detectCapability(block, 'securityHeaders'),
    accessLog: detectCapability(block, 'accessLog'),
    errorPage: detectCapability(block, 'errorPage'),
    tls: detectTls(block),
    otherImports: collectOtherImports(block),
  };
}

/** 能力徽章文案（服务目录卡片 / 抽屉使用） */
export function describeSiteCapabilities(
  snapshot: SiteCapabilitiesSnapshot,
): Array<{ key: string; label: string; status: SiteCapabilityStatus }> {
  return [
    { key: 'waf', label: 'WAF', status: snapshot.waf.status },
    { key: 'geo', label: 'IP 限制', status: snapshot.geo.status },
    { key: 'securityHeaders', label: '安全头', status: snapshot.securityHeaders.status },
    { key: 'errorPage', label: '错误页', status: snapshot.errorPage.status },
    { key: 'compression', label: '压缩', status: snapshot.compression.status },
    { key: 'accessLog', label: '日志', status: snapshot.accessLog.status },
  ];
}

function buildRemovePatterns(key: SiteCapabilityKey): RegExp[] {
  const anchor = CAPABILITY_ANCHORS[key];
  const patterns = [importLineRegex(anchor.importName)];
  if (key === 'compression') patterns.push(/^[ \t]*encode\s/);
  return patterns;
}

/**
 * 把能力行写回单个站点块：先删该能力当前占有的行，再把新行插到块头之后。
 *
 * 其余字节一概不动（注释、缩进、顺序全部保留）；lines 为空数组表示关闭该能力。
 * 目标行已存在时不做任何改动，保证"已开启的重复开启"字节级不变、连续应用幂等。
 */
export function applyCapabilityToSiteBlock(
  siteBlock: string,
  key: SiteCapabilityKey,
  lines: string[],
): string {
  if (!siteBlock) return siteBlock;

  const elements = siteBlock.split('\n');
  const headerIdx = findHeaderLine(elements);
  if (headerIdx < 0) return siteBlock;

  const indent = detectChildIndent(elements, headerIdx);
  const crlfSuffix = siteBlock.includes('\r\n') ? '\r' : '';
  const desired = lines.map(line => `${indent}${line}${crlfSuffix}`);
  // 目标行已按期望形态存在（含缩进），直接原样返回，避免移动既有行的位置。
  if (
    desired.length > 0 &&
    desired.every(want => elements.some(line => line === want))
  ) {
    return siteBlock;
  }

  const removePatterns = buildRemovePatterns(key);
  const parts: string[] = [];
  for (let idx = 0; idx < elements.length; idx += 1) {
    const line = elements[idx]!;
    if (idx === headerIdx) {
      parts.push(line);
      parts.push(...desired);
      continue;
    }
    const probe = `${line}\n`;
    if (removePatterns.some(pattern => pattern.test(probe))) continue;
    parts.push(line);
  }
  return parts.join('\n');
}

/** 各能力 render：把开启意图转成要写进站点块的指令行。 */
export function renderCapabilityLines(
  key: SiteCapabilityKey,
  value: { enabled?: boolean; name?: string } | undefined,
): string[] {
  const anchor = CAPABILITY_ANCHORS[key];
  if (!value?.enabled) return [];
  if (key === 'accessLog') {
    const name = value.name?.trim() || 'site';
    return [`import ${anchor.importName} ${name}`];
  }
  return [`import ${anchor.importName}`];
}

/** 站点块头里的第一个 host token 取域名前缀，作为日志文件名兜底 */
function deriveLogNameFromHeader(headerLine: string): string {
  const token = (headerLine ?? '').trim().split(/\s+/)[0] ?? '';
  const host = token.replace(/^https?:\/\//, '').split('/')[0] ?? '';
  const prefix = host.split(':')[0]?.split('.')[0] ?? '';
  return prefix && !prefix.startsWith('(') ? prefix : '';
}

/**
 * 从行数组里摘掉所有 import subdomain-log：单行写法删一行，块写法删到配对的 }。
 *
 * 新写法与旧写法可能同时存在（用户手工改过），两种都清干净再统一写回，
 * 避免留下两份互相覆盖的日志配置。
 */
function stripSiteLogImports(elements: string[]): string[] {
  const kept: string[] = [];
  for (let idx = 0; idx < elements.length; idx += 1) {
    const line = elements[idx] ?? '';
    if (!SITE_LOG_IMPORT_HEAD_RE.test(line)) {
      kept.push(line);
      continue;
    }
    // 单行写法到此为止；块写法数花括号跳过整块（块参数本身不含嵌套括号）
    let depth = countBraces(line);
    while (depth > 0 && idx + 1 < elements.length) {
      idx += 1;
      depth += countBraces(elements[idx] ?? '');
    }
  }
  return kept;
}

/** 行内花括号净增量：{ 加一、} 减一 */
function countBraces(line: string): number {
  let depth = 0;
  for (const char of line) {
    if (char === '{') depth += 1;
    else if (char === '}') depth -= 1;
  }
  return depth;
}

/**
 * 把站点日志三开关写回单个站点块。
 *
 * 名字优先沿用块内既有 import 的参数，其次用调用方给的名字，最后回落到
 * 块头域名前缀，避免仅改开关时把日志文件名改掉、把历史日志切成两份。
 * 两开关全关时不写任何行（level OFF 不是合法指令），只做删除。
 * 其余字节（注释、缩进、顺序）一概不动，连续应用幂等。
 */
export function applySiteLogToSiteBlock(
  siteBlock: string,
  settings: SiteLogSettings,
  name?: string,
): string {
  if (!siteBlock) return siteBlock;

  const elements = siteBlock.split('\n');
  const headerIdx = findHeaderLine(elements);
  if (headerIdx < 0) return siteBlock;

  const existing = parseSiteLogImport(siteBlock);
  const resolvedName =
    existing?.name ||
    name?.trim() ||
    deriveLogNameFromHeader(elements[headerIdx] ?? '') ||
    'site';

  const indent = detectChildIndent(elements, headerIdx);
  // 块内续行比指令行再深一级，沿用站点既有缩进（Tab 站点不混入空格缩进）
  const argIndent = indent.includes('\t') ? `${indent}\t` : `${indent}  `;
  const crlfSuffix = siteBlock.includes('\r\n') ? '\r' : '';
  const desired = renderSiteImportBlock(settings, resolvedName).map((line) => {
    if (!line.startsWith('  ')) return `${indent}${line}${crlfSuffix}`;
    return `${argIndent}${line.slice(2)}${crlfSuffix}`;
  });

  // 已按期望形态存在（且没有残留的旧写法）时原样返回，避免移动既有行的位置
  const stripped = stripSiteLogImports(elements);
  const unchanged = elements.length === stripped.length;
  if (unchanged && desired.every((want) => elements.includes(want))) {
    return siteBlock;
  }

  const base = unchanged ? elements : stripped;
  // 删除后块头行号可能前移，重新定位
  const targetIdx = unchanged ? headerIdx : findHeaderLine(base);
  if (targetIdx < 0) return siteBlock;

  const parts = base.slice();
  parts.splice(targetIdx + 1, 0, ...desired);
  return parts.join('\n');
}
