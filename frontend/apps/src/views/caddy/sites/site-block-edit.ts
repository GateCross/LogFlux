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
  upstream?: string;
  tlsMode?: EditableTlsMode;
  /** manual 模式的证书/私钥路径（Caddy 可读的容器内路径） */
  tlsCertPath?: string;
  tlsKeyPath?: string;
}

function trailingCr(line: string | undefined): string {
  return line?.endsWith('\r') ? '\r' : '';
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
