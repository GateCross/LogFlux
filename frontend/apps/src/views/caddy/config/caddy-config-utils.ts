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
