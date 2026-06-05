import type { CaddyFormModel } from './types';
import { genId } from './caddy-config-utils';

export type DiffRow = {
  left: string | null;
  right: string | null;
  type: 'same' | 'added' | 'removed' | 'changed';
  leftNo: number | null;
  rightNo: number | null;
  key: string;
};

export function buildLineDiff(leftRaw: string, rightRaw: string): DiffRow[] {
  const left = leftRaw.split('\n');
  const right = rightRaw.split('\n');
  const m = left.length;
  const n = right.length;
  const dp: number[][] = Array.from({ length: m + 1 }, () => Array(n + 1).fill(0));
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (left[i - 1] === right[j - 1]) {
        dp[i]![j] = dp[i - 1]![j - 1]! + 1;
      } else {
        dp[i]![j] = Math.max(dp[i - 1]![j]!, dp[i]![j - 1]!);
      }
    }
  }
  const ops: Array<{ left: string | null; right: string | null; type: 'same' | 'added' | 'removed' }> = [];
  let i = m;
  let j = n;
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && left[i - 1] === right[j - 1]) {
      ops.push({ left: left[i - 1]!, right: right[j - 1]!, type: 'same' });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || dp[i]![j - 1]! >= dp[i - 1]![j]!)) {
      ops.push({ left: null, right: right[j - 1]!, type: 'added' });
      j--;
    } else if (i > 0) {
      ops.push({ left: left[i - 1]!, right: null, type: 'removed' });
      i--;
    }
  }
  ops.reverse();

  const rows: Array<{ left: string | null; right: string | null; type: 'same' | 'added' | 'removed' | 'changed' }> = [];
  let k = 0;
  while (k < ops.length) {
    const current = ops[k]!;
    const next = ops[k + 1];
    if (current.type === 'removed' && next?.type === 'added') {
      rows.push({ left: current.left, right: next.right, type: 'changed' });
      k += 2;
      continue;
    }
    rows.push({ left: current.left, right: current.right, type: current.type });
    k += 1;
  }

  let leftLine = 0;
  let rightLine = 0;
  return rows.map((row, index) => {
    if (row.left !== null) leftLine += 1;
    if (row.right !== null) rightLine += 1;
    return {
      ...row,
      leftNo: row.left !== null ? leftLine : null,
      rightNo: row.right !== null ? rightLine : null,
      key: `${index}-${row.type}`
    };
  });
}

export function formatCaddyfile(content: string): string {
  if (!content.trim()) return content;
  const lines = content.split('\n');
  const out: string[] = [];
  let indent = 0;
  const indentUnit = '  ';

  const stripComment = (line: string) => {
    let inQuote = false;
    let escaped = false;
    let result = '';
    for (const ch of line) {
      if (!escaped && ch === '"') inQuote = !inQuote;
      if (!inQuote && ch === '#') break;
      result += ch;
      escaped = ch === '\\' && !escaped;
      if (ch !== '\\') escaped = false;
    }
    return result;
  };

  const countBraces = (line: string) => {
    let openCount = 0;
    let closeCount = 0;
    let inQuote = false;
    let escaped = false;
    const sanitized = stripComment(line);
    const isBoundary = (ch?: string) => !ch || /\s/.test(ch);
    for (let idx = 0; idx < sanitized.length; idx += 1) {
      const ch = sanitized[idx];
      if (!escaped && ch === '"') inQuote = !inQuote;
      if (!inQuote && (ch === '{' || ch === '}')) {
        const prev = sanitized[idx - 1];
        const next = sanitized[idx + 1];
        if (isBoundary(prev) && isBoundary(next)) {
          if (ch === '{') openCount += 1;
          if (ch === '}') closeCount += 1;
        }
      }
      escaped = ch === '\\' && !escaped;
      if (ch !== '\\') escaped = false;
    }
    return { openCount, closeCount };
  };

  for (const raw of lines) {
    const trimmed = raw.trim();
    if (!trimmed) {
      out.push('');
      continue;
    }
    const { openCount, closeCount } = countBraces(raw);
    const nextIndent = Math.max(indent - closeCount, 0);
    out.push(`${indentUnit.repeat(nextIndent)}${trimmed}`);
    indent = nextIndent + openCount;
  }
  return out.join('\n').trim();
}

export function normalizeModules(raw: any): CaddyFormModel {
  const globalRaw = typeof raw?.global?.raw === 'string' ? raw.global.raw : '';
  const normalized: CaddyFormModel = {
    schemaVersion: raw.schemaVersion ?? 1,
    global: { ...(raw.global ?? {}), raw: globalRaw },
    upstreams: (raw.upstreams ?? []).map((u: any) => ({
      name: u.name || `upstream-${genId().slice(0, 6)}`,
      targets: Array.isArray(u.targets) ? u.targets : [],
      lbPolicy: u.lbPolicy ?? 'round_robin',
      healthCheck: u.healthCheck
    })),
    sites: (raw.sites ?? []).map((s: any) => ({
      id: s.id || genId(),
      name: s.name || '未命名站点',
      enabled: s.enabled ?? true,
      domains: Array.isArray(s.domains) ? s.domains : [],
      tls: s.tls ?? { mode: 'auto' },
      imports: s.imports ?? [],
      geoip2Vars: s.geoip2Vars ?? [],
      encode: s.encode ?? [],
      headers: s.headers,
      routes: (s.routes ?? []).map((r: any) => ({
        id: r.id || genId(),
        name: r.name || '未命名路由',
        enabled: r.enabled ?? true,
        match: {
          host: r.match?.host ?? [],
          path: r.match?.path ?? [],
          method: r.match?.method ?? [],
          header: r.match?.header ?? [],
          query: r.match?.query ?? [],
          expression: r.match?.expression ?? ''
        },
        logAppend: r.logAppend ?? [],
        handles: (r.handles ?? []).map((h: any) => ({
          id: h.id || genId(),
          type: h.type || 'reverse_proxy',
          enabled: h.enabled ?? true,
          upstream: h.upstream ?? '',
          lbPolicy: h.lbPolicy ?? 'round_robin',
          healthCheck: h.healthCheck,
          transportProtocol: h.transportProtocol ?? '',
          tlsInsecureSkipVerify: h.tlsInsecureSkipVerify ?? false,
          root: h.root,
          browse: h.browse,
          status: h.status,
          body: h.body,
          to: h.to,
          code: h.code,
          rules: h.rules ?? [],
          uri: h.uri
        }))
      }))
    }))
  };
  return normalized;
}
