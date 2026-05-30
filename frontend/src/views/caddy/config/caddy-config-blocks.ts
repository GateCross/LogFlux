import type { CaddyBlockDraft, CaddyFormModel, PreservedCaddyBlock, Site } from './types';
import { buildCaddyfile, parseCaddyfileToModules } from './caddy-config-parser';
import { normalizeModules } from './caddy-config-diff';
import { analyzeSiteForQuickConfig } from './quick-config-utils';
import { genId, isSiteToken } from './caddy-config-utils';

export function createPreservedBlock(raw: string, reason: string, kind: PreservedCaddyBlock['kind'] = 'unknown'): PreservedCaddyBlock {
  const firstLine = raw.split('\n').find(line => line.trim().length > 0) ?? '';
  // 取 Caddyfile 块首行作为摘要标题；多域名时只取第一个
  const rawTitle = firstLine.replace(/\{.*$/, '').trim();
  const firstDomain = rawTitle.includes(',') ? rawTitle.split(',')[0].trim() : rawTitle;
  const title = firstDomain.length > 60 ? `${firstDomain.slice(0, 60)}...` : firstDomain || '(未命名块)';
  return { id: genId(), kind, title, raw, reason };
}

export function isEditableSite(site: Site, upstreamNames?: Set<string>): boolean {
  const names = upstreamNames ?? new Set<string>();
  return analyzeSiteForQuickConfig(site, names).kind === 'simple';
}

// parser 能识别的站点级 directive 关键词
const KNOWN_SITE_DIRECTIVES = new Set([
  'reverse_proxy', 'file_server', 'respond', 'redir', 'rewrite', 'header',
  'handle', 'import', 'geoip2_vars', 'encode', 'tls', 'root', 'log_append'
]);

/**
 * 检测站点原始 Caddyfile 文本中是否含有 parser 未处理的 directive。
 * parser 会忽略未知指令，导致 round-trip 丢数据；有未知指令的站点必须整体保留。
 */
function siteHasUnknownDirectives(raw: string): boolean {
  const lines = raw.split('\n');
  for (const line of lines) {
    const trimmed = line.replace(/#.*/, '').trim();
    if (!trimmed || trimmed === '{' || trimmed === '}') continue;
    // 跳过 matcher 定义行
    if (trimmed.startsWith('@')) continue;
    // 跳过 handle 子块内的闭合
    if (trimmed.startsWith('transport ') || trimmed.includes('tls_insecure_skip_verify')) continue;

    const firstWord = trimmed.split(/\s+/)[0];
    if (!firstWord) continue;
    // 纯花括号行
    if (firstWord === '{' || firstWord === '}') continue;
    // 域名行（站点头）
    if (firstWord.includes('.') || firstWord.startsWith(':')) continue;

    if (!KNOWN_SITE_DIRECTIVES.has(firstWord)) {
      return true;
    }
  }
  return false;
}

/**
 * 从原始 Caddyfile 解析为分块草稿：
 * - 简单站点进入 sites（可编辑）
 * - 复杂站点进入 preservedBlocks（只读保留）
 * - 全局块、snippet、无法识别块也进入 preservedBlocks
 */
export function parseCaddyfileToBlocks(config: string): CaddyBlockDraft {
  const modules = parseCaddyfileToModules(config);
  const upstreamNames = new Set(modules.upstreams.map(u => u.name).filter(Boolean));

  const editableSites: Site[] = [];
  const preservedBlocks: PreservedCaddyBlock[] = [];

  // 从原始配置中提取站点块文本，用于 preserved 原样保留
  const siteBlockTexts = extractSiteBlocks(config);

  for (const site of modules.sites) {
    const raw = siteBlockTexts.get(site.domains[0] ?? '') ?? buildSiteRaw(site);

    // 原始 Caddyfile 中有 parser 未识别的 directive，必须整体保留以防丢失
    if (siteHasUnknownDirectives(raw)) {
      preservedBlocks.push(createPreservedBlock(raw, '包含 parser 未识别的指令，整体保留', 'site'));
      continue;
    }

    if (isEditableSite(site, upstreamNames)) {
      editableSites.push(site);
    } else {
      const { kind, summary } = analyzeSiteForQuickConfig(site, upstreamNames);
      const reason = kind === 'complex' ? summary.reasons.join('；') : '无法解析为简单站点';
      preservedBlocks.push(createPreservedBlock(raw, reason, 'site'));
    }
  }

  // 全局配置中若有无法结构化的片段也保留，并从 globalRaw 中移除以避免重复
  const globalRaw = modules.global?.raw ?? '';
  let strippedGlobalRaw = globalRaw;
  if (globalRaw.trim()) {
    const snippetBlocks = extractSnippetsAndUnknown(globalRaw);
    if (snippetBlocks.length > 0) {
      // 从全局配置中移除已提取的块，防止 buildCaddyfileFromBlocks 时重复拼接
      strippedGlobalRaw = stripExtractedBlocks(globalRaw, snippetBlocks);
      for (const block of snippetBlocks) {
        preservedBlocks.push(block);
      }
    }
  }

  // 独立 snippet（不在全局块内，如 (common_headers)）也必须保留
  const standaloneSnippets = extractStandaloneSnippets(config);
  const existingRawSet = new Set(preservedBlocks.map(b => b.raw.trim()));
  for (const block of standaloneSnippets) {
    if (!existingRawSet.has(block.raw.trim())) {
      preservedBlocks.push(block);
    }
  }

  const normalized = normalizeModules({
    schemaVersion: 1,
    global: { ...modules.global, raw: strippedGlobalRaw },
    upstreams: modules.upstreams,
    sites: editableSites
  });

  return { ...normalized, preservedBlocks };
}

/**
 * 将分块草稿合并为完整 Caddyfile：
 * - 可编辑站点生成结构化 Caddyfile
 * - preservedBlocks 原样拼接
 */
export function buildCaddyfileFromBlocks(draft: CaddyBlockDraft): string {
  const model: CaddyFormModel = {
    schemaVersion: draft.schemaVersion,
    global: draft.global,
    upstreams: draft.upstreams,
    sites: draft.sites
  };

  const parts: string[] = [];

  const globalRaw = model.global?.raw?.trim();
  if (globalRaw) {
    parts.push(globalRaw);
  }

  // snippet 必须位于引用它的站点之前，否则 Caddy 会把 import 当作文件导入。
  if (draft.preservedBlocks?.length) {
    for (const block of draft.preservedBlocks.filter(item => item.kind !== 'site')) {
      if (block.raw.trim()) {
        parts.push(block.raw.trim());
      }
    }
  }

  // 可编辑站点生成 Caddyfile（不重复输出 global.raw）
  const structuredCaddyfile = buildCaddyfile({ ...model, global: { ...model.global, raw: '' } });
  if (structuredCaddyfile && structuredCaddyfile !== '# No sites defined' && structuredCaddyfile !== '# No routes defined') {
    parts.push(structuredCaddyfile);
  }

  // 复杂站点最后拼接，保证它们 import 的片段已先定义。
  if (draft.preservedBlocks?.length) {
    for (const block of draft.preservedBlocks.filter(item => item.kind === 'site')) {
      if (block.raw.trim()) {
        parts.push(block.raw.trim());
      }
    }
  }

  const result = parts.join('\n\n').trim();
  return result || '# No sites defined';
}

/** 从原始 Caddyfile 文本中提取站点块，以首个域名做 key */
function extractSiteBlocks(config: string): Map<string, string> {
  const result = new Map<string, string>();
  const lines = config.split('\n');
  let depth = 0;
  let currentBlock: string[] = [];
  let currentKey = '';

  for (const line of lines) {
    const trimmed = line.trim();
    const openCount = (line.match(/{/g) || []).length;
    const closeCount = (line.match(/}/g) || []).length;

    if (depth === 0 && openCount > 0 && !trimmed.startsWith('{') && !trimmed.startsWith('(')) {
      const before = trimmed.split('{')[0].trim();
      const tokens = before.replace(/,/g, ' ').split(/\s+/).filter(Boolean);
      // 复用 isSiteToken 保持与 parseCaddyfileToModules 一致的判断标准
      const hasSiteToken = tokens.some(t => isSiteToken(t));
      if (hasSiteToken) {
        currentKey = tokens[0];
        currentBlock = [line];
        depth += openCount - closeCount;
        continue;
      }
    }

    if (depth > 0) {
      currentBlock.push(line);
      depth += openCount - closeCount;
      if (depth <= 0) {
        depth = 0;
        if (currentKey) {
          result.set(currentKey, currentBlock.join('\n'));
        }
        currentBlock = [];
        currentKey = '';
      }
    }
  }

  return result;
}

/** 从全局配置中提取 snippet 和无法识别的块 */
function extractSnippetsAndUnknown(globalRaw: string): PreservedCaddyBlock[] {
  const blocks: PreservedCaddyBlock[] = [];
  const lines = globalRaw.split('\n');
  let depth = 0;
  let currentBlock: string[] = [];
  let inBlock = false;

  for (const line of lines) {
    const trimmed = line.trim();
    const openCount = (line.match(/{/g) || []).length;
    const closeCount = (line.match(/}/g) || []).length;

    if (depth === 0 && openCount > 0) {
      inBlock = true;
      currentBlock = [line];
      depth += openCount - closeCount;
      continue;
    }

    if (inBlock) {
      currentBlock.push(line);
      depth += openCount - closeCount;
      if (depth <= 0) {
        depth = 0;
        inBlock = false;
        const raw = currentBlock.join('\n');
        const firstLine = currentBlock[0]?.trim() ?? '';
        const kind: PreservedCaddyBlock['kind'] = firstLine.startsWith('(') ? 'snippet' : 'global';
        blocks.push(createPreservedBlock(raw, '全局配置块无法结构化编辑', kind));
        currentBlock = [];
      }
    }
  }

  return blocks;
}

/**
 * 从全局配置文本中移除已被提取的花括号块，
 * 保留非块行（如顶层指令 `order` 等）。
 */
function stripExtractedBlocks(globalRaw: string, _extracted: PreservedCaddyBlock[]): string {
  const lines = globalRaw.split('\n');
  const kept: string[] = [];
  let depth = 0;
  let inBlock = false;

  for (const line of lines) {
    const openCount = (line.match(/{/g) || []).length;
    const closeCount = (line.match(/}/g) || []).length;

    if (depth === 0 && openCount > 0) {
      // 进入一个花括号块，跳过（已被提取到 preservedBlocks）
      inBlock = true;
      depth += openCount - closeCount;
      continue;
    }

    if (inBlock) {
      depth += openCount - closeCount;
      if (depth <= 0) {
        depth = 0;
        inBlock = false;
      }
      continue;
    }

    // 非块行保留（如 `order ...` 等顶层指令）
    kept.push(line);
  }

  return kept.join('\n').trim();
}

/**
 * 从完整 Caddyfile 中提取独立 snippet 块（不在全局选项块内，也不在站点块内）。
 * 例如出现在全局块和站点块之间的 (common_headers) { ... }。
 */
function extractStandaloneSnippets(config: string): PreservedCaddyBlock[] {
  const blocks: PreservedCaddyBlock[] = [];
  const lines = config.split('\n');
  let depth = 0;
  let currentBlock: string[] = [];
  let inBlock = false;
  let isSnippet = false;

  for (const line of lines) {
    const trimmed = line.trim();
    const openCount = (line.match(/{/g) || []).length;
    const closeCount = (line.match(/}/g) || []).length;

    if (depth === 0 && openCount > 0) {
      // 判断是否为 snippet 块（以 ( 开头）
      if (trimmed.startsWith('(')) {
        inBlock = true;
        isSnippet = true;
        currentBlock = [line];
        depth += openCount - closeCount;
        continue;
      }
      // 非 snippet 块（站点或全局），跳过整块
      inBlock = true;
      isSnippet = false;
      currentBlock = [];
      depth += openCount - closeCount;
      continue;
    }

    if (inBlock) {
      if (isSnippet) currentBlock.push(line);
      depth += openCount - closeCount;
      if (depth <= 0) {
        depth = 0;
        inBlock = false;
        if (isSnippet && currentBlock.length > 0) {
          const raw = currentBlock.join('\n');
          blocks.push(createPreservedBlock(raw, '独立 snippet 片段', 'snippet'));
        }
        isSnippet = false;
        currentBlock = [];
      }
    }
  }

  return blocks;
}

/** 将 Site 对象转回简化 Caddyfile 文本（兜底用） */
function buildSiteRaw(site: Site): string {
  const lines: string[] = [];
  lines.push(`${site.domains.join(' ')} {`);
  for (const route of site.routes) {
    for (const h of route.handles) {
      if (h.type === 'reverse_proxy') lines.push(`  reverse_proxy ${h.upstream ?? 'localhost:8080'}`);
      else if (h.type === 'file_server') lines.push(`  file_server`);
      else if (h.type === 'redirect') lines.push(`  redir ${h.to ?? '/'} ${h.code ?? 302}`);
      else lines.push(`  # ${h.type}`);
    }
  }
  lines.push('}');
  return lines.join('\n');
}
