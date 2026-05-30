import type { CaddyFormModel } from './types';
import { siteDomainRe } from './caddy-config-utils';

const portOnlyRe = /^:\d+$/;
const methodAllowList = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];

const isValidPathPattern = (value: string) => {
  if (!value) return false;
  if (value.startsWith('/')) return true;
  if (value.startsWith('*')) return true;
  if (value.startsWith('{') && value.endsWith('}')) return true;
  return false;
};

export function validateStructuredConfig(model: CaddyFormModel, hasPreservedContent?: boolean): string[] {
  const errors: string[] = [];
  const pushError = (errorMessage: string) => {
    errors.push(errorMessage);
  };

  const enabledSites = model.sites.filter(s => s.enabled);
  const hasSites = enabledSites.length > 0;
  const hasGlobalRaw = Boolean(model.global?.raw?.trim());
  if (!hasSites && !hasGlobalRaw && !hasPreservedContent) {
    pushError('至少需要一个站点或全局配置');
  }

  const upstreamNames = new Set<string>();
  for (const up of model.upstreams) {
    if (!up.name) pushError('上游名称不能为空');
    if (upstreamNames.has(up.name)) pushError(`上游名称重复: ${up.name}`);
    upstreamNames.add(up.name);
    if (up.targets.length === 0) pushError(`上游 ${up.name} 至少配置一个目标`);
  }

  for (const site of model.sites) {
    if (!site.enabled) continue;
    if (!site.name) pushError('站点名称不能为空');
    if (site.domains.length === 0) pushError(`站点 ${site.name || site.id} 至少配置一个域名`);

    const hasEnabledRoutes = site.routes.some(route => route.enabled);
    const hasImports = (site.imports ?? []).some(item => item.trim().length > 0);
    if (!hasEnabledRoutes && !hasImports) {
      pushError(`站点 ${site.name || site.id} 至少配置一个路由或 import`);
    }

    const invalidDomains = site.domains.filter(d => d && !(siteDomainRe.test(d) || portOnlyRe.test(d)));
    if (invalidDomains.length) pushError(`站点 ${site.name || site.id} 域名格式不合法: ${invalidDomains.join(', ')}`);

    if (site.tls?.mode === 'manual' && (!site.tls.certFile || !site.tls.keyFile)) {
      pushError(`站点 ${site.name || site.id} TLS 手动模式需填写证书和私钥`);
    }

    for (const route of site.routes) {
      if (!route.enabled) continue;
      if (!route.name) pushError(`站点 ${site.name || site.id} 有未命名路由`);
      if (route.handles.length === 0) pushError(`路由 ${route.name || route.id} 至少一个 Handler`);
      if (route.handles.every(h => !h.enabled)) {
        pushError(`路由 ${route.name || route.id} 至少启用一个 Handler`);
      }

      const invalidPaths = route.match.path.filter(p => p && !isValidPathPattern(p));
      if (invalidPaths.length) pushError(`路由 ${route.name || route.id} Path 格式不合法: ${invalidPaths.join(', ')}`);

      const invalidMethods = route.match.method.filter(m => m && !methodAllowList.includes(m.toUpperCase()));
      if (invalidMethods.length) pushError(`路由 ${route.name || route.id} Method 非法: ${invalidMethods.join(', ')}`);

      for (const handle of route.handles) {
        if (!handle.enabled) continue;
        if (handle.type === 'reverse_proxy' && !handle.upstream) {
          pushError(`路由 ${route.name || route.id} 的 reverse_proxy 未选择上游`);
        }
      }
    }
  }

  return errors;
}
