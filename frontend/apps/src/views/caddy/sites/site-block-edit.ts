/**
 * 站点块基本字段的外科编辑（纯函数）。
 *
 * 只改目标行（块头 / reverse_proxy / tls），其余字节不动；与能力写回
 * （applyCapabilityToSiteBlock）同一哲学：定点替换，不重建全文。
 * 仅用于简单站点（单路由单反代）；复杂站点在抽屉里只读 + 能力开关。
 */

import { detectChildIndent, findHeaderLine } from '../config/caddy-config-utils';

export type EditableTlsMode = 'auto' | 'dns' | 'internal' | 'manual' | 'off';

export interface BasicSiteEdits {
  domains?: string[];
  /** 上游目标（可含 https:// 前缀），写入首条 reverse_proxy */
  upstream?: string;
  /** 上游跳过 TLS 验证：维护 reverse_proxy 块内 transport http { tls_insecure_skip_verify } */
  upstreamSkipVerify?: boolean;
  tlsMode?: EditableTlsMode;
  /** manual 模式的证书/私钥路径（Caddy 可读的容器内路径） */
  tlsCertPath?: string;
  tlsKeyPath?: string;
}

function trailingCr(line: string | undefined): string {
  return line?.endsWith('\r') ? '\r' : '';
}

/** 块内新插入行是否需要补 \r（CRLF 块保持行尾风格） */
function blockCrlf(block: string): string {
  return block.includes('\r\n') ? '\r' : '';
}

/** 一行的大括号净值（{ 为正、} 为负），用于定位指令块边界 */
function braceDelta(line: string | undefined): number {
  const open = (line ?? '').match(/\{/g)?.length ?? 0;
  const close = (line ?? '').match(/\}/g)?.length ?? 0;
  return open - close;
}

/**
 * 首条 reverse_proxy 指令的范围：head 行下标与其子块结束行下标（无子块时 end === head）。
 * reverse_proxy 可嵌在 handle 等子块内，从块头之后全量扫描。
 */
function findReverseProxyRange(
  lines: string[],
  headerIdx: number,
): { head: number; end: number } | null {
  for (let idx = headerIdx + 1; idx < lines.length; idx += 1) {
    const raw = lines[idx] ?? '';
    if (!/^\s*reverse_proxy(\s|\{)/.test(raw)) continue;
    if (!raw.trimEnd().endsWith('{')) return { head: idx, end: idx };
    let depth = 0;
    let end = idx;
    for (; end < lines.length; end += 1) {
      depth += braceDelta(lines[end]);
      if (end > idx && depth <= 0) break;
    }
    return { head: idx, end };
  }
  return null;
}

/**
 * 在 reverse_proxy 指令范围内开启/关闭「跳过上游 TLS 验证」：
 * 开启 → reverse_proxy 变块形式并插入 transport http { tls_insecure_skip_verify }；
 * 关闭 → 删 skip 行，transport 子块与 reverse_proxy 块因此变空时一并收回。
 * 已处于目标状态时字节级不变（幂等）。
 */
function applyUpstreamSkipVerify(
  lines: string[],
  range: { head: number; end: number },
  enable: boolean,
  crlf: string,
): void {
  const headRaw = lines[range.head] ?? '';
  const headLead = (headRaw.match(/^[ \t]*/) ?? [''])[0];
  const childIndent = `${headLead}  `;

  // 定位既有 transport http 子块与 skip 行
  let transportHead = -1;
  let transportEnd = -1;
  let skipLine = -1;
  let transportOtherLines = 0;
  if (range.end > range.head) {
    let inTransport = false;
    let depth = 0;
    for (let idx = range.head + 1; idx <= range.end; idx += 1) {
      const raw = lines[idx] ?? '';
      if (!inTransport && /^[ \t]*transport\s+http(\s|\{)/.test(raw)) {
        inTransport = true;
        transportHead = idx;
        depth = braceDelta(raw);
        continue;
      }
      if (!inTransport) continue;
      depth += braceDelta(raw);
      if (depth <= 0) {
        transportEnd = idx;
        break;
      }
      if (/^[ \t]*tls_insecure_skip_verify\s*$/.test(raw)) {
        skipLine = idx;
      } else if (raw.trim()) {
        transportOtherLines += 1;
      }
    }
  }

  if (enable) {
    if (skipLine >= 0) return;
    if (transportHead >= 0) {
      const skipIndent =
        ((lines[transportHead] ?? '').match(/^[ \t]*/) ?? [''])[0] + '  ';
      lines.splice(transportHead + 1, 0, `${skipIndent}tls_insecure_skip_verify${crlf}`);
      return;
    }
    if (range.end > range.head) {
      // reverse_proxy 已有子块：transport 子块插到块头之后，原子行整体后移
      lines.splice(
        range.head + 1,
        0,
        `${childIndent}transport http {${crlf}`,
        `${childIndent}  tls_insecure_skip_verify${crlf}`,
        `${childIndent}}${crlf}`,
      );
      return;
    }
    // 无子块：reverse_proxy 行转为块形式，插入 transport 子块后补配对闭括号
    lines.splice(
      range.head,
      1,
      `${headRaw.trimEnd()} {${crlf}`,
      `${childIndent}transport http {${crlf}`,
      `${childIndent}  tls_insecure_skip_verify${crlf}`,
      `${childIndent}}${crlf}`,
      `${headLead}}${crlf}`,
    );
    return;
  }

  if (skipLine < 0) return;
  if (transportOtherLines === 0 && transportHead >= 0 && transportEnd >= 0) {
    // skip 是 transport 子块唯一内容：整块收回（先于 skip 单行删除，索引未被移动）
    lines.splice(transportHead, transportEnd - transportHead + 1);
  } else {
    lines.splice(skipLine, 1);
  }
  // reverse_proxy 块因此变空（transport 是唯一子内容且紧跟块头）则收回块形式；
  // range.end 是 reverse_proxy 自身闭括号，transport 块尾应紧邻其前
  if (
    transportOtherLines === 0 &&
    transportHead === range.head + 1 &&
    transportEnd === range.end - 1 &&
    range.end > range.head
  ) {
    lines.splice(
      range.head,
      2,
      `${headRaw.trimEnd().replace(/\s*\{$/, '')}${crlf}`,
    );
  }
}

/**
 * 应用基本字段编辑：域名改块头、上游改首条 reverse_proxy、TLS 改 tls 行/import。
 * 无匹配可改的输入原样返回；改完保持原有缩进、子块结构与行尾风格。
 */
export function applyBasicEditsToSiteBlock(block: string, edits: BasicSiteEdits): string {
  if (!block) return block;
  const lines = block.split('\n');
  const headerIdx = findHeaderLine(lines);
  if (headerIdx < 0) return block;
  const indent = detectChildIndent(lines, headerIdx);

  const domains = (edits.domains ?? []).map((item) => item.trim()).filter(Boolean);
  if (domains.length > 0) {
    lines[headerIdx] = `${domains.join(' ')} {${trailingCr(lines[headerIdx])}`;
  }

  if (edits.upstream?.trim()) {
    const target = edits.upstream.trim();
    for (let idx = headerIdx + 1; idx < lines.length; idx += 1) {
      const raw = lines[idx] ?? '';
      if (!/^\s*reverse_proxy\s/.test(raw)) continue;
      const lead = (raw.match(/^[ \t]*/) ?? [''])[0] || indent;
      const hadBlock = raw.trimEnd().endsWith('{');
      lines[idx] = `${lead}reverse_proxy ${target}${hadBlock ? ' {' : ''}${trailingCr(raw)}`;
      break;
    }
  }

  if (edits.upstreamSkipVerify !== undefined) {
    const range = findReverseProxyRange(lines, headerIdx);
    // 无 reverse_proxy 指令：跳过验证无处安放，保持原样（不产生孤立 transport 行）
    if (range) {
      applyUpstreamSkipVerify(lines, range, edits.upstreamSkipVerify, blockCrlf(block));
    }
  }

  if (edits.tlsMode) {
    const manualCertPath = edits.tlsMode === 'manual' ? edits.tlsCertPath?.trim() : '';
    const manualKeyPath = edits.tlsMode === 'manual' ? edits.tlsKeyPath?.trim() : '';
    // manual 路径不齐全时整个 TLS 编辑不动：半行 tls 指令过不了 adapt，
    // 先删旧行又拿不出新行会让站点静默失去 HTTPS
    if (edits.tlsMode === 'manual' && (!manualCertPath || !manualKeyPath)) {
      return lines.join('\n');
    }
    // tls 行与 dns import 互斥：先两清，再按目标模式插回块头之后；
    // tls 块（tls { ... }）须整块删到配对闭括号，只删头行会留下孤儿子行
    let idx = headerIdx + 1;
    while (idx < lines.length) {
      const raw = lines[idx] ?? '';
      if (/^[ \t]*tls(\s|\{)/.test(raw)) {
        if (raw.trimEnd().endsWith('{')) {
          let depth = 0;
          let end = idx;
          for (; end < lines.length; end += 1) {
            const text = lines[end] ?? '';
            depth += (text.match(/\{/g) ?? []).length;
            depth -= (text.match(/\}/g) ?? []).length;
            if (end > idx && depth <= 0) break;
          }
          lines.splice(idx, end - idx + 1);
        } else {
          lines.splice(idx, 1);
        }
        continue;
      }
      if (/^[ \t]*import\s+cloudflare/.test(raw)) {
        lines.splice(idx, 1);
        continue;
      }
      idx += 1;
    }
    if (edits.tlsMode === 'dns') {
      lines.splice(headerIdx + 1, 0, `${indent}import cloudflare`);
    } else if (edits.tlsMode === 'manual') {
      lines.splice(headerIdx + 1, 0, `${indent}tls ${manualCertPath} ${manualKeyPath}`);
    } else if (edits.tlsMode === 'internal' || edits.tlsMode === 'off') {
      lines.splice(headerIdx + 1, 0, `${indent}tls ${edits.tlsMode}`);
    }
    // auto：删掉既有行即回退自动签发
  }

  return lines.join('\n');
}

/**
 * 用新块替换配置里第一次出现的旧块。
 * ponytail: 首次出现即替换，站点块文本重复的概率可忽略；真撞上时再按行号定位。
 */
export function replaceSiteBlockInConfig(config: string, oldBlock: string, newBlock: string): string {
  if (!config.includes(oldBlock)) return config;
  // 须用函数替换：新块里的 $&、$' 等替换模式串按字面写入，不能被当模式展开
  return config.replace(oldBlock, () => newBlock);
}

/**
 * 解析块内既有手动证书路径（tls <cert> <key> 双路径形态）；无匹配返回 null。
 */
export function parseManualTlsPaths(
  block: string,
): { certPath: string; keyPath: string } | null {
  const match = (block ?? '').match(/^[ \t]*tls\s+(\S+)\s+(\S+)\s*$/m);
  if (!match?.[1] || !match[2]) return null;
  return { certPath: match[1], keyPath: match[2] };
}

/**
 * 解析首条 reverse_proxy 的上游目标（可含 https:// 前缀，不含子块）。
 * 无 reverse_proxy 或目标是块形式占位（如 `reverse_proxy {`）时返回 null。
 */
export function parseReverseProxyTarget(block: string): string | null {
  const match = (block ?? '').match(/^[ \t]*reverse_proxy\s+(.+?)\s*\{?\s*$/m);
  const target = match?.[1]?.trim() ?? '';
  if (!target || target === '{') return null;
  return target;
}

/**
 * 判断首条 reverse_proxy 是否启用了「跳过上游 TLS 验证」：
 * 指令块内存在 transport http 子块且其中包含 tls_insecure_skip_verify 行。
 */
export function parseUpstreamSkipVerify(block: string): boolean {
  const lines = (block ?? '').split('\n');
  const headerIdx = findHeaderLine(lines);
  const range = findReverseProxyRange(lines, headerIdx);
  if (!range) return false;
  let inTransport = false;
  let depth = 0;
  for (let idx = range.head; idx <= range.end; idx += 1) {
    const raw = lines[idx] ?? '';
    if (!inTransport && /^[ \t]*transport\s+http(\s|\{)/.test(raw)) {
      inTransport = true;
      depth = braceDelta(raw);
      continue;
    }
    if (!inTransport) continue;
    depth += braceDelta(raw);
    if (depth <= 0) break;
    if (/^[ \t]*tls_insecure_skip_verify\s*$/.test(raw)) return true;
  }
  return false;
}
