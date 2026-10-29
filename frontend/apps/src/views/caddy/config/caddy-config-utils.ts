export const siteDomainRe = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$/;
export const siteIpv4Re = /^(?:\d{1,3}\.){3}\d{1,3}$/;

export function genId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function isSiteToken(token: string): boolean {
  if (!token) return false;
  if (token.startsWith('(') || token.endsWith(')') || token.includes('(') || token.includes(')')) return false;
  if (token.startsWith(':') && /^\:\d+$/.test(token)) return true;
  const [host = '', port] = token.split(':');
  if (port) {
    if (!/^\d+$/.test(port)) return false;
    return siteDomainRe.test(host) || siteIpv4Re.test(host) || host === 'localhost';
  }
  return siteDomainRe.test(token);
}

/**
 * 上游地址的 path 校验错误；合法返回空串。
 * Caddy 上游只接受 scheme + host + port，带 path/query 会在 /adapt 阶段被拒；
 * unix/ 与 unix+h2c/ 后面的绝对路径属于合法网络地址形式，不算路径。
 */
export function upstreamPathError(value: string): string {
  const addr = value.trim();
  if (!addr || /^unix(\+h2c)?\//.test(addr)) return '';
  const rest = addr.replace(/^[a-z][a-z0-9+.-]*:\/\//i, '');
  return /[/?]/.test(rest)
    ? `上游地址不支持路径: ${addr}（Caddy 只接受 scheme://host:port，前缀请用 handle_path 单独配置）`
    : '';
}

/** 找到以 { 结尾的块头行行号；找不到返回 -1。 */
export function findHeaderLine(lines: string[]): number {
  for (let idx = 0; idx < lines.length; idx += 1) {
    if ((lines[idx] ?? '').trimEnd().endsWith('{')) return idx;
  }
  return -1;
}

/** 取块内子行缩进：从块头下一行起沿用首条非空且非闭括号的行，兜底两空格。 */
export function detectChildIndent(lines: string[], headerIdx: number): string {
  for (let idx = headerIdx + 1; idx < lines.length; idx += 1) {
    const line = lines[idx] ?? '';
    const text = line.trim();
    if (text === '' || text === '}') continue;
    return (line.match(/^[ \t]*/) ?? [''])[0] || '  ';
  }
  return '  ';
}

// Re-export from sub-modules for backward compatibility
export { validateStructuredConfig } from './caddy-config-validator';
export { buildLineDiff, formatCaddyfile, normalizeModules, type DiffRow } from './caddy-config-diff';
export { buildCaddyfile, parseCaddyfileToModules } from './caddy-config-parser';
