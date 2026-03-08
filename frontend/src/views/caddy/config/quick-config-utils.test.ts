import assert from 'node:assert/strict';
import test from 'node:test';

import type { CaddyFormModel, Site } from './types';
import {
  analyzeSiteForQuickConfig,
  buildQuickConfigState,
  buildSiteFromQuickDraft,
  mergeQuickConfigDrafts
} from './quick-config-utils';

function createProxySite(partial?: Partial<Site>): Site {
  return {
    id: partial?.id ?? 'site-1',
    name: partial?.name ?? 'proxy-site',
    enabled: partial?.enabled ?? true,
    domains: partial?.domains ?? ['example.com'],
    tls: partial?.tls ?? { mode: 'auto' },
    imports: partial?.imports ?? [],
    geoip2Vars: partial?.geoip2Vars ?? [],
    encode: partial?.encode ?? [],
    headers: partial?.headers,
    routes: partial?.routes ?? [
      {
        id: 'route-1',
        name: '默认路由',
        enabled: true,
        match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
        logAppend: [],
        handles: [
          {
            id: 'handle-1',
            type: 'reverse_proxy',
            enabled: true,
            upstream: '127.0.0.1:8080',
            lbPolicy: 'round_robin',
            transportProtocol: '',
            tlsInsecureSkipVerify: false
          }
        ]
      }
    ]
  };
}

function createFormModel(sites: Site[]): CaddyFormModel {
  return {
    schemaVersion: 1,
    global: { raw: '' },
    upstreams: [],
    sites
  };
}

test('analyzeSiteForQuickConfig marks simple reverse proxy site as editable', () => {
  const site = createProxySite();
  const result = analyzeSiteForQuickConfig(site, new Set());

  assert.equal(result.kind, 'simple');
  if (result.kind === 'simple') {
    assert.equal(result.draft.mode, 'reverse_proxy');
    assert.equal(result.draft.upstream, '127.0.0.1:8080');
  }
});

test('analyzeSiteForQuickConfig marks matcher site as complex', () => {
  const site = createProxySite({
    routes: [
      {
        id: 'route-1',
        name: '默认路由',
        enabled: true,
        match: { host: [], path: ['/api/*'], method: [], header: [], query: [], expression: '' },
        logAppend: [],
        handles: [
          {
            id: 'handle-1',
            type: 'reverse_proxy',
            enabled: true,
            upstream: '127.0.0.1:8080',
            lbPolicy: 'round_robin',
            transportProtocol: '',
            tlsInsecureSkipVerify: false
          }
        ]
      }
    ]
  });

  const result = analyzeSiteForQuickConfig(site, new Set());
  assert.equal(result.kind, 'complex');
  if (result.kind === 'complex') {
    assert.ok(result.summary.reasons.some(item => item.includes('matcher')));
  }
});

test('buildSiteFromQuickDraft supports static and redirect modes', () => {
  const staticSite = buildSiteFromQuickDraft({
    id: 'site-static',
    name: 'static',
    enabled: true,
    domains: ['static.example.com'],
    tlsMode: 'internal',
    mode: 'file_server',
    upstream: '',
    root: '/srv/www',
    browse: true,
    redirectTo: '',
    redirectCode: 302
  });

  const redirectSite = buildSiteFromQuickDraft({
    id: 'site-redirect',
    name: 'redirect',
    enabled: true,
    domains: ['old.example.com'],
    tlsMode: 'off',
    mode: 'redirect',
    upstream: '',
    root: '',
    browse: false,
    redirectTo: 'https://new.example.com',
    redirectCode: 308
  });

  assert.equal(staticSite.routes[0].handles[0].type, 'file_server');
  assert.equal(staticSite.routes[0].handles[0].root, '/srv/www');
  assert.equal(redirectSite.routes[0].handles[0].type, 'redirect');
  assert.equal(redirectSite.routes[0].handles[0].code, 308);
});

test('mergeQuickConfigDrafts updates simple sites and preserves complex sites', () => {
  const simpleSite = createProxySite({ id: 'simple-site', name: 'simple-site' });
  const complexSite = createProxySite({
    id: 'complex-site',
    name: 'complex-site',
    routes: [
      {
        id: 'route-1',
        name: '默认路由',
        enabled: true,
        match: { host: [], path: ['/api/*'], method: [], header: [], query: [], expression: '' },
        logAppend: [],
        handles: [
          {
            id: 'handle-1',
            type: 'reverse_proxy',
            enabled: true,
            upstream: '127.0.0.1:8080',
            lbPolicy: 'round_robin',
            transportProtocol: '',
            tlsInsecureSkipVerify: false
          }
        ]
      }
    ]
  });
  const formModel = createFormModel([simpleSite, complexSite]);
  const quickState = buildQuickConfigState(formModel);

  assert.equal(quickState.simpleSites.length, 1);
  assert.equal(quickState.complexSites.length, 1);

  const merged = mergeQuickConfigDrafts(formModel, [
    {
      ...quickState.simpleSites[0],
      upstream: '127.0.0.1:9090'
    },
    {
      id: 'new-site',
      name: 'new-site',
      enabled: true,
      domains: ['new.example.com'],
      tlsMode: 'auto',
      mode: 'redirect',
      upstream: '',
      root: '',
      browse: false,
      redirectTo: 'https://new.example.com',
      redirectCode: 308
    }
  ]);

  assert.equal(merged.sites.length, 3);
  assert.equal(merged.sites[0].routes[0].handles[0].upstream, '127.0.0.1:9090');
  assert.equal(merged.sites[1].routes[0].match.path[0], '/api/*');
  assert.equal(merged.sites[2].routes[0].handles[0].type, 'redirect');
});
