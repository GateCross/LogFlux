import type { CaddyFormModel, Route, RouteMatch, Site } from './types';
import { genId, isSiteToken } from './caddy-config-utils';
import { normalizeModules } from './caddy-config-diff';

function extractGlobalOptionsBlock(content: string): { raw: string; rest: string } {
  const lines = content.split('\n');
  const globalLines: string[] = [];
  const restLines: string[] = [];
  let depth = 0;
  let currentBlock: 'global' | 'site' | null = null;

  for (const line of lines) {
    const trimmed = line.trim();
    const sanitized = line.replace(/#.*/, '');
    const openCount = (sanitized.match(/{/g) || []).length;
    const closeCount = (sanitized.match(/}/g) || []).length;

    if (depth === 0 && openCount > 0) {
      const before = (sanitized.split('{')[0] ?? '').trim();
      let blockType: 'global' | 'site' = 'global';
      if (trimmed.startsWith('{')) {
        blockType = 'global';
      } else if (before.startsWith('(')) {
        blockType = 'global';
      } else {
        const tokens = before.replace(/,/g, ' ').split(/\s+/).filter(Boolean);
        const hasSiteToken = tokens.some(t => isSiteToken(t));
        blockType = hasSiteToken ? 'site' : 'global';
      }
      currentBlock = blockType;
    }

    if (depth === 0 && openCount === 0 && currentBlock === null) {
      if (trimmed.length > 0) {
        globalLines.push(line);
      } else {
        restLines.push(line);
      }
      continue;
    }

    if (currentBlock === 'site') {
      restLines.push(line);
    } else {
      globalLines.push(line);
    }

    depth += openCount - closeCount;
    if (depth <= 0) {
      depth = 0;
      currentBlock = null;
    }
  }

  return {
    raw: globalLines.join('\n').trim(),
    rest: restLines.join('\n')
  };
}

export function parseCaddyfileToModules(content: string): CaddyFormModel {
  const { raw: globalRaw, rest } = extractGlobalOptionsBlock(content);
  const sites: Site[] = [];
  const matchers: Record<string, RouteMatch> = {};
  const lines = rest.split('\n');
  let depth = 0;
  let currentSite: Site | null = null;
  let currentRoute: Route | null = null;
  let currentHandleBlock = false;
  let handleDepth: number | null = null;
  let currentMatcherName: string | null = null;
  let matcherDepth: number | null = null;
  let reverseProxyDepth: number | null = null;
  let currentReverseProxy: any | null = null;
  let currentSiteRoot = '';

  function ensureDefaultRoute() {
    if (!currentSite) return;
    if (!currentRoute) {
      currentRoute = {
        id: genId(),
        name: '默认路由',
        enabled: true,
        match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
        logAppend: [],
        handles: []
      };
      currentSite.routes.push(currentRoute);
    }
  }

  function cloneMatcher(matcher?: RouteMatch): RouteMatch {
    if (!matcher) return { host: [], path: [], method: [], header: [], query: [], expression: '' };
    return {
      host: [...(matcher.host ?? [])],
      path: [...(matcher.path ?? [])],
      method: [...(matcher.method ?? [])],
      header: [...(matcher.header ?? [])],
      query: [...(matcher.query ?? [])],
      expression: matcher.expression ?? ''
    };
  }

  function createRouteForMatcher(matcherName?: string): Route | null {
    if (!currentSite) return null;
    if (!matcherName) {
      ensureDefaultRoute();
      return currentRoute;
    }
    const match = cloneMatcher(matchers[matcherName]);
    const route: Route = {
      id: genId(),
      name: `@${matcherName}`,
      enabled: true,
      match,
      logAppend: [],
      handles: []
    };
    currentSite.routes.push(route);
    return route;
  }

  for (const raw of lines) {
    const line = raw.replace(/#.*/, '').trim();
    if (!line) continue;
    const openCount = (line.match(/{/g) || []).length;
    const closeCount = (line.match(/}/g) || []).length;
    if (reverseProxyDepth !== null && depth >= reverseProxyDepth) {
      if (line.startsWith('transport ') && currentReverseProxy) {
        const proto = line.replace('transport ', '').replace('{', '').trim();
        if (proto) currentReverseProxy.transportProtocol = proto;
      }
      if (line.includes('tls_insecure_skip_verify') && currentReverseProxy) {
        currentReverseProxy.tlsInsecureSkipVerify = true;
      }
      depth += openCount - closeCount;
      if (depth < reverseProxyDepth) {
        reverseProxyDepth = null;
        currentReverseProxy = null;
      }
      continue;
    }
    if (depth === 0 && line.includes('{') && !line.startsWith('{')) {
      const before = (line.split('{')[0] ?? '').trim();
      if (before) {
        const domains = before.replace(/,/g, ' ').split(/\s+/).filter(Boolean).filter(isSiteToken);
        const firstDomain = domains[0];
        if (firstDomain) {
          const site: Site = {
            id: genId(),
            name: firstDomain,
            enabled: true,
            domains,
            imports: [],
            geoip2Vars: [],
            encode: [],
            tls: { mode: 'auto' },
            routes: []
          };
          currentSite = site;
          sites.push(site);
          currentSiteRoot = '';
          currentRoute = null;
          currentHandleBlock = false;
          handleDepth = null;
          currentMatcherName = null;
          matcherDepth = null;
          reverseProxyDepth = null;
          currentReverseProxy = null;
        }
      }
    }

    if (currentSite) {
      // matcher block: @m { host ... }
      if (line.startsWith('@') && line.includes('{')) {
        const name = (line.split('{')[0] ?? '').trim().slice(1);
        if (name) {
          matchers[name] = { host: [], path: [], method: [], header: [], query: [], expression: '' };
          currentMatcherName = name;
          matcherDepth = depth + openCount;
        }
      } else if (currentMatcherName) {
        const matcher = matchers[currentMatcherName];
        if (!matcher) continue;
        if (line.startsWith('host ')) matcher.host = line.replace('host ', '').split(/\s+/).filter(Boolean);
        if (line.startsWith('path ')) matcher.path = line.replace('path ', '').split(/\s+/).filter(Boolean);
        if (line.startsWith('method ')) matcher.method = line.replace('method ', '').split(/\s+/).filter(Boolean);
        if (line.startsWith('header ')) {
          const parts = line.replace('header ', '').split(/\s+/);
          if (parts.length >= 2 && parts[0]) matcher.header.push({ key: parts[0], value: parts.slice(1).join(' ') });
        }
        if (line.startsWith('query ')) {
          const parts = line.replace('query ', '').split(/\s+/);
          parts.forEach(q => {
            const [k, v] = q.split('=');
            if (k) matcher.query.push({ key: k, value: v ?? '' });
          });
        }
        if (line.startsWith('expression ')) {
          matcher.expression = line.replace('expression ', '').trim();
        }
      }

      if (!currentHandleBlock && !currentMatcherName) {
        if (line.startsWith('import ')) {
          currentSite.imports = currentSite.imports || [];
          const value = line.replace('import ', '').trim();
          if (value) currentSite.imports.push(value);
        } else if (line.startsWith('geoip2_vars ')) {
          currentSite.geoip2Vars = currentSite.geoip2Vars || [];
          const value = line.replace('geoip2_vars ', '').trim();
          if (value) currentSite.geoip2Vars.push(value);
        } else if (line.startsWith('encode ')) {
          const enc = line.replace('encode ', '').trim().split(/\s+/).filter(Boolean);
          currentSite.encode = enc;
        } else if (line.startsWith('tls') && !line.startsWith('tls_insecure')) {
          const parts = line.split(/\s+/).filter(Boolean);
          if (parts.length === 1) {
            currentSite.tls = { mode: 'auto' };
          } else if (parts[1] === 'off') {
            currentSite.tls = { mode: 'off' };
          } else if (parts[1] === 'internal') {
            currentSite.tls = { mode: 'internal' };
          } else if (parts.length >= 3) {
            currentSite.tls = { mode: 'manual', certFile: parts[1], keyFile: parts[2] };
          }
        } else if (line.startsWith('root ')) {
          const parts = line.split(/\s+/).filter(Boolean);
          if (parts.length >= 3) {
            let idx = 1;
            if (parts[idx] === '*' || parts[idx]?.startsWith('@')) idx += 1;
            const rootPath = parts.slice(idx).join(' ');
            if (rootPath) {
              currentSiteRoot = rootPath;
              for (const route of currentSite.routes) {
                for (const h of route.handles) {
                  if (h.type === 'file_server' && !h.root) {
                    h.root = rootPath;
                  }
                }
              }
            }
          }
        } else if (line.startsWith('file_server')) {
          const parts = line.split(/\s+/).filter(Boolean);
          const matcherName = parts[1]?.startsWith('@') ? parts[1].slice(1) : '';
          const browse = parts.includes('browse');
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'file_server',
              enabled: true,
              root: currentSiteRoot || undefined,
              browse
            });
          }
        } else if (line.startsWith('reverse_proxy ')) {
          const parts = line.split(/\s+/).filter(Boolean);
          let matcherName = '';
          let idx = 1;
          if (parts[1]?.startsWith('@')) {
            matcherName = parts[1].slice(1);
            idx = 2;
          }
          const rawTargets = parts.slice(idx);
          const targets: string[] = [];
          for (const t of rawTargets) {
            if (t === '{') break;
            if (t.endsWith('{')) {
              const cleaned = t.slice(0, -1);
              if (cleaned) targets.push(cleaned);
              break;
            }
            targets.push(t);
          }
          const route = createRouteForMatcher(matcherName);
          if (route) {
            const handle = {
              id: genId(),
              type: 'reverse_proxy' as const,
              enabled: true,
              upstream: targets.join(' ').trim(),
              transportProtocol: '',
              tlsInsecureSkipVerify: false
            };
            route.handles.push(handle);
            if (line.includes('{')) {
              reverseProxyDepth = depth + openCount;
              currentReverseProxy = handle;
            }
          }
        } else if (line.startsWith('respond ')) {
          const rest = line.replace('respond ', '').trim();
          const parts = rest.split(/\s+/);
          let matcherName = '';
          if (parts[0]?.startsWith('@')) {
            matcherName = parts.shift()!.slice(1);
          }
          const payload = parts.join(' ');
          const match = payload.match(/^"?(.*?)"?\s+(\d+)?$/);
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'respond',
              enabled: true,
              body: match?.[1] ?? '',
              status: match?.[2] ? Number(match[2]) : 200
            });
          }
        } else if (line.startsWith('redir ')) {
          const parts = line.replace('redir ', '').trim().split(/\s+/);
          let matcherName = '';
          if (parts[0]?.startsWith('@')) {
            matcherName = parts.shift()!.slice(1);
          }
          let code: number | undefined;
          if (parts[1]) {
            if (parts[1] === 'permanent') code = 308;
            else if (parts[1] === 'temporary') code = 302;
            else if (!Number.isNaN(Number(parts[1]))) code = Number(parts[1]);
          }
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'redirect',
              enabled: true,
              to: parts[0] ?? '/',
              code: code ?? 302
            });
          }
        } else if (line.startsWith('rewrite ')) {
          const parts = line.replace('rewrite ', '').trim().split(/\s+/);
          let matcherName = '';
          if (parts[0]?.startsWith('@')) {
            matcherName = parts.shift()!.slice(1);
          }
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'rewrite',
              enabled: true,
              uri: parts[1] ?? parts[0]
            });
          }
        } else if (line.startsWith('header ')) {
          let rest = line.replace('header ', '').trim();
          const tokens = rest.split(/\s+/);
          let matcherName = '';
          if (tokens[0]?.startsWith('@')) {
            matcherName = tokens.shift()!.slice(1);
            rest = tokens.join(' ');
          }
          const isDelete = rest.startsWith('-');
          const kv = rest.replace(/^-/, '').split(/\s+/);
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'header',
              enabled: true,
              rules: [{ op: isDelete ? 'delete' : 'set', key: kv[0] ?? '', value: kv.slice(1).join(' ') }]
            });
          }
        } else if (line.startsWith('log_append ')) {
          const parts = line.replace('log_append ', '').trim().split(/\s+/);
          let matcherName = '';
          if (parts[0]?.startsWith('@')) {
            matcherName = parts.shift()!.slice(1);
          }
          if (parts[0]) {
            const route = createRouteForMatcher(matcherName);
            if (route) {
              route.logAppend = route.logAppend || [];
              route.logAppend.push({ key: parts[0], value: parts.slice(1).join(' ') });
            }
          }
        }
      }

      if (line.startsWith('handle ') || line === 'handle {') {
        currentRoute = {
          id: genId(),
          name: line.startsWith('handle ') ? line.replace('handle', '').replace('{', '').trim() : '默认路由',
          enabled: true,
          match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
          logAppend: [],
          handles: []
        };
        currentSite.routes.push(currentRoute);
        currentHandleBlock = true;
        handleDepth = depth + openCount;
        const matchName = line.startsWith('handle ')
          ? line.replace('handle', '').replace('{', '').trim().replace(/^@/, '')
          : '';
        if (matchName && currentRoute) {
          const m = matchers[matchName];
          if (m) currentRoute.match = m;
        }
      }

      if (currentHandleBlock && currentRoute) {
        if (line.startsWith('reverse_proxy ')) {
          const rawTargets = line.replace('reverse_proxy ', '').trim().split(/\s+/).filter(Boolean);
          const targets: string[] = [];
          for (const t of rawTargets) {
            if (t === '{') break;
            if (t.endsWith('{')) {
              const cleaned = t.slice(0, -1);
              if (cleaned) targets.push(cleaned);
              break;
            }
            targets.push(t);
          }
          const handle = {
            id: genId(),
            type: 'reverse_proxy' as const,
            enabled: true,
            upstream: targets.join(' ').trim(),
            transportProtocol: '',
            tlsInsecureSkipVerify: false
          };
          currentRoute.handles.push(handle);
          if (line.includes('{')) {
            reverseProxyDepth = depth + openCount;
            currentReverseProxy = handle;
          }
        } else if (line.startsWith('file_server')) {
          currentRoute.handles.push({
            id: genId(),
            type: 'file_server',
            enabled: true,
            browse: line.includes('browse')
          });
        } else if (line.startsWith('respond ')) {
          const parts = line.replace('respond ', '').match(/^"?(.*?)"?\s+(\d+)?$/);
          currentRoute.handles.push({
            id: genId(),
            type: 'respond',
            enabled: true,
            body: parts?.[1] ?? '',
            status: parts?.[2] ? Number(parts[2]) : 200
          });
        } else if (line.startsWith('redir ')) {
          const parts = line.replace('redir ', '').split(/\s+/);
          let code: number | undefined;
          if (parts[1]) {
            if (parts[1] === 'permanent') code = 308;
            else if (parts[1] === 'temporary') code = 302;
            else if (!Number.isNaN(Number(parts[1]))) code = Number(parts[1]);
          }
          currentRoute.handles.push({
            id: genId(),
            type: 'redirect',
            enabled: true,
            to: parts[0] ?? '/',
            code: code ?? 302
          });
        } else if (line.startsWith('rewrite ')) {
          const parts = line.replace('rewrite ', '').split(/\s+/);
          currentRoute.handles.push({
            id: genId(),
            type: 'rewrite',
            enabled: true,
            uri: parts[1] ?? parts[0]
          });
        } else if (line.startsWith('header ')) {
          const rest = line.replace('header ', '').trim();
          const isDelete = rest.startsWith('-');
          const kv = rest.replace(/^-/, '').split(/\s+/);
          currentRoute.handles.push({
            id: genId(),
            type: 'header',
            enabled: true,
            rules: [{ op: isDelete ? 'delete' : 'set', key: kv[0] ?? '', value: kv.slice(1).join(' ') }]
          });
        } else if (line.startsWith('log_append ')) {
          const parts = line.replace('log_append ', '').trim().split(/\s+/);
          if (parts[0]) {
            currentRoute.logAppend = currentRoute.logAppend || [];
            currentRoute.logAppend.push({ key: parts[0], value: parts.slice(1).join(' ') });
          }
        }
      }
    }

    depth += openCount - closeCount;
    if (depth < 0) depth = 0;
    if (matcherDepth !== null && depth < matcherDepth) {
      matcherDepth = null;
      currentMatcherName = null;
    }
    if (handleDepth !== null && depth < handleDepth) {
      handleDepth = null;
      currentHandleBlock = false;
      currentRoute = null;
    }
  }
  return normalizeModules({
    schemaVersion: 1,
    global: { raw: globalRaw },
    upstreams: [],
    sites
  });
}

export function buildCaddyfile(
  model: CaddyFormModel,
  options?: { includeDisabled?: boolean; includeGlobal?: boolean }
): string {
  const lines: string[] = [];
  const globalRaw = model.global?.raw?.trim();
  if (globalRaw && options?.includeGlobal !== false) {
    lines.push(globalRaw);
    lines.push('');
  }
  if (!model.sites || model.sites.length === 0) {
    return lines.length ? lines.join('\n').trim() : '# No sites defined';
  }
  const upstreamMap = new Map(model.upstreams.map(u => [u.name, u]));
  const includeDisabled = options?.includeDisabled ?? false;
  for (const site of model.sites.filter(s => includeDisabled || s.enabled)) {
    const usedMatcherNames = new Set<string>();
    const hosts = site.domains.join(' ');
    if (!hosts) continue;
    lines.push(`${hosts} {`);
    if (site.geoip2Vars?.length) {
      site.geoip2Vars.forEach(v => {
        if (v) lines.push(`  geoip2_vars ${v}`);
      });
    }
    if (site.imports?.length) {
      site.imports.forEach(v => {
        if (v) lines.push(`  import ${v}`);
      });
    }
    if (site.encode?.length) {
      lines.push(`  encode ${site.encode.join(' ')}`);
    }
    if (site.tls?.mode) {
      if (site.tls.mode === 'off') lines.push(`  tls off`);
      else if (site.tls.mode === 'internal') lines.push(`  tls internal`);
      else if (site.tls.mode === 'manual' && site.tls.certFile && site.tls.keyFile) {
        lines.push(`  tls ${site.tls.certFile} ${site.tls.keyFile}`);
      }
    }
    for (const route of site.routes.filter(r => includeDisabled || r.enabled)) {
      const matcherLines: string[] = [];
      if (route.match.host.length) matcherLines.push(`host ${route.match.host.join(' ')}`);
      if (route.match.path.length) matcherLines.push(`path ${route.match.path.join(' ')}`);
      if (route.match.method.length) matcherLines.push(`method ${route.match.method.join(' ')}`);
      if (route.match.header.length) {
        matcherLines.push(`header ${route.match.header.map(h => `${h.key} ${h.value}`).join(' ')}`);
      }
      if (route.match.query.length) {
        matcherLines.push(`query ${route.match.query.map(q => `${q.key}=${q.value}`).join(' ')}`);
      }
      if (route.match.expression) {
        matcherLines.push(`expression ${route.match.expression}`);
      }

      let matcherName = '';
      const rawRouteName = route.name?.trim() ?? '';
      if (rawRouteName.startsWith('@')) {
        const token = rawRouteName.slice(1);
        if (/^[a-zA-Z0-9_-]+$/.test(token)) matcherName = rawRouteName;
      }
      if (!matcherName) {
        const base = `@m_${route.id.slice(0, 6)}`;
        matcherName = base;
        let idx = 1;
        while (usedMatcherNames.has(matcherName)) {
          matcherName = `${base}_${idx}`;
          idx += 1;
        }
      }
      if (matcherLines.length) {
        usedMatcherNames.add(matcherName);
      }

      const enabledHandles = route.handles.filter(hd => includeDisabled || hd.enabled);
      const headerOnly =
        enabledHandles.length > 0 && enabledHandles.every(h => h.type === 'header') && !route.logAppend?.length;
      const fileServerOnly =
        enabledHandles.length > 0 && enabledHandles.every(h => h.type === 'file_server') && !route.logAppend?.length;

      if (matcherLines.length) {
        lines.push(`  ${matcherName} {`);
        matcherLines.forEach(l => lines.push(`    ${l}`));
        lines.push(`  }`);
        if (headerOnly) {
          for (const h of enabledHandles) {
            for (const r of h.rules ?? []) {
              if (r.op === 'delete') lines.push(`  header ${matcherName} -${r.key}`);
              else lines.push(`  header ${matcherName} ${r.key} ${r.value ?? ''}`.replace(/\s+$/, ''));
            }
          }
          continue;
        }
        lines.push(`  handle ${matcherName} {`);
      } else if (headerOnly) {
        for (const h of enabledHandles) {
          for (const r of h.rules ?? []) {
            if (r.op === 'delete') lines.push(`  header -${r.key}`);
            else lines.push(`  header ${r.key} ${r.value ?? ''}`.replace(/\s+$/, ''));
          }
        }
        continue;
      } else if (fileServerOnly) {
        for (const h of enabledHandles) {
          if (h.root) lines.push(`  root * ${h.root}`);
          lines.push(`  file_server${h.browse ? ' browse' : ''}`);
        }
        continue;
      } else {
        lines.push(`  handle {`);
      }

      for (const h of enabledHandles) {
        if (h.type === 'reverse_proxy') {
          const up = h.upstream ? upstreamMap.get(h.upstream) : undefined;
          const targets = up?.targets.length ? up.targets.join(' ') : h.upstream ? h.upstream : 'localhost:8080';
          const transport = h.transportProtocol || (h.tlsInsecureSkipVerify ? 'http' : '');
          if (transport || h.tlsInsecureSkipVerify) {
            lines.push(`    reverse_proxy ${targets} {`);
            if (transport) {
              lines.push(`      transport ${transport} {`);
              if (h.tlsInsecureSkipVerify) lines.push(`        tls_insecure_skip_verify`);
              lines.push(`      }`);
            }
            lines.push(`    }`);
          } else {
            lines.push(`    reverse_proxy ${targets}`);
          }
        } else if (h.type === 'file_server') {
          if (h.root) lines.push(`    root * ${h.root}`);
          lines.push(`    file_server${h.browse ? ' browse' : ''}`);
        } else if (h.type === 'respond') {
          const body = (h.body ?? '').replaceAll('"', '\\"');
          lines.push(`    respond "${body}" ${h.status ?? 200}`);
        } else if (h.type === 'redirect') {
          const code = h.code ?? 302;
          const codeStr = code === 308 ? 'permanent' : code === 302 ? 'temporary' : String(code);
          lines.push(`    redir ${h.to ?? '/'} ${codeStr}`);
        } else if (h.type === 'header') {
          for (const r of h.rules ?? []) {
            if (r.op === 'delete') lines.push(`    header -${r.key}`);
            else lines.push(`    header ${r.key} ${r.value ?? ''}`.replace(/\s+$/, ''));
          }
        } else if (h.type === 'rewrite') {
          lines.push(`    rewrite * ${h.uri ?? '/'}`);
        }
      }
      if (route.logAppend?.length) {
        for (const item of route.logAppend) {
          if (!item.key) continue;
          lines.push(`    log_append ${item.key} ${item.value ?? ''}`.replace(/\s+$/, ''));
        }
      }

      lines.push(`  }`);
    }
    lines.push(`}`);
    lines.push('');
  }
  const result = lines.join('\n').trim();
  return result || '# No routes defined';
}
