import assert from 'node:assert/strict';
import test from 'node:test';
import type { CaddyBlockDraft, Site } from './types';
import {
  buildCaddyfileFromBlocks,
  createPreservedBlock,
  isEditableSite,
  parseCaddyfileToBlocks
} from './caddy-config-blocks';

test('parseCaddyfileToBlocks: 简单反代站点解析为可编辑站点', () => {
  const config = `example.com {
  reverse_proxy 127.0.0.1:8080
}`;
  const draft = parseCaddyfileToBlocks(config);

  assert.equal(draft.sites.length, 1);
  assert.equal(draft.sites[0].domains[0], 'example.com');
  assert.equal(draft.sites[0].routes[0].handles[0].type, 'reverse_proxy');
  assert.equal(draft.sites[0].routes[0].handles[0].upstream, '127.0.0.1:8080');
  assert.equal(draft.preservedBlocks.length, 0);
});

test('parseCaddyfileToBlocks: 静态站点和重定向站点解析为可编辑站点', () => {
  const config = `static.example.com {
  root * /srv/www
  file_server
}

old.example.com {
  redir https://new.example.com permanent
}`;
  const draft = parseCaddyfileToBlocks(config);

  assert.equal(draft.sites.length, 2);
  assert.equal(draft.sites[0].domains[0], 'static.example.com');
  assert.equal(draft.sites[0].routes[0].handles[0].type, 'file_server');
  assert.equal(draft.sites[1].domains[0], 'old.example.com');
  assert.equal(draft.sites[1].routes[0].handles[0].type, 'redirect');
  assert.equal(draft.preservedBlocks.length, 0);
});

test('parseCaddyfileToBlocks: 包含未知 directive 的站点进入 preservedBlocks', () => {
  const config = `complex.example.com {
  reverse_proxy 127.0.0.1:8080
  unknown_directive foo bar
}`;
  const draft = parseCaddyfileToBlocks(config);

  // 含未知指令的站点必须归入 preservedBlocks，不能进入可编辑 sites
  assert.equal(draft.sites.length, 0, '含未知指令的站点不应出现在可编辑 sites 中');
  const siteBlocks = draft.preservedBlocks.filter(b => b.kind === 'site');
  assert.equal(siteBlocks.length, 1, '应有 1 个 preserved site block');
  assert.ok(siteBlocks[0].raw.includes('unknown_directive'), 'preserved raw 应包含未知指令原文');
  assert.ok(siteBlocks[0].raw.includes('reverse_proxy'), 'preserved raw 应包含已知指令原文');
});

test('parseCaddyfileToBlocks: snippet 和复杂全局块原样保留', () => {
  const config = `(common_headers) {
  header X-Frame-Options SAMEORIGIN
  header X-Content-Type-Options nosniff
}

example.com {
  reverse_proxy 127.0.0.1:8080
}`;
  const draft = parseCaddyfileToBlocks(config);

  assert.equal(draft.sites.length, 1);
  // snippet 应在 preservedBlocks 中保留
  const snippetBlocks = draft.preservedBlocks.filter(b => b.kind === 'snippet');
  assert.ok(snippetBlocks.length >= 1);
  assert.ok(snippetBlocks[0].raw.includes('common_headers'));
  assert.ok(snippetBlocks[0].raw.includes('X-Frame-Options'));
});

test('parseCaddyfileToBlocks: snippet 被提取后从 global.raw 中移除，避免重复', () => {
  const config = `(common_headers) {
  header X-Frame-Options SAMEORIGIN
}

order header_before root

example.com {
  reverse_proxy 127.0.0.1:8080
}`;
  const draft = parseCaddyfileToBlocks(config);

  // snippet 应在 preservedBlocks 中
  const snippetBlocks = draft.preservedBlocks.filter(b => b.kind === 'snippet');
  assert.equal(snippetBlocks.length, 1);

  // global.raw 中不应再包含 snippet 内容
  const globalRaw = draft.global?.raw ?? '';
  assert.ok(!globalRaw.includes('common_headers'), 'global.raw 应移除已提取的 snippet');

  // 但非块行（如 order 指令）应保留在 global.raw 中
  assert.ok(globalRaw.includes('order header_before root'), 'global.raw 应保留非块顶层指令');

  // 生成的 Caddyfile 中 snippet 只出现一次
  const caddyfile = buildCaddyfileFromBlocks(draft);
  const snippetOccurrences = (caddyfile.match(/common_headers/g) || []).length;
  assert.equal(snippetOccurrences, 1, 'snippet 在生成的 Caddyfile 中应只出现一次');
});

test('buildCaddyfileFromBlocks: preserved blocks 内容不变', () => {
  const snippetRaw = `(custom_block) {
  header Custom-Header value
}`;

  const siteRaw = `complex.example.com {
  reverse_proxy 127.0.0.1:9090
  unknown_directive test
}`;

  const draft: CaddyBlockDraft = {
    schemaVersion: 1,
    global: { raw: '' },
    upstreams: [],
    sites: [
      {
        id: 's1',
        name: 'simple',
        enabled: true,
        domains: ['simple.example.com'],
        tls: { mode: 'auto' },
        imports: [],
        geoip2Vars: [],
        encode: [],
        routes: [
          {
            id: 'r1',
            name: '默认路由',
            enabled: true,
            match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
            logAppend: [],
            handles: [
              {
                id: 'h1',
                type: 'reverse_proxy' as const,
                enabled: true,
                upstream: '127.0.0.1:8080',
                lbPolicy: 'round_robin' as const,
                tlsInsecureSkipVerify: false
              }
            ]
          }
        ]
      }
    ],
    preservedBlocks: [
      createPreservedBlock(snippetRaw, '测试 snippet', 'snippet'),
      createPreservedBlock(siteRaw, '复杂站点', 'site')
    ]
  };

  const caddyfile = buildCaddyfileFromBlocks(draft);

  // 可编辑站点生成了内容
  assert.ok(caddyfile.includes('simple.example.com'));
  assert.ok(caddyfile.includes('reverse_proxy 127.0.0.1:8080'));

  // preserved blocks 原样包含
  assert.ok(caddyfile.includes('custom_block'));
  assert.ok(caddyfile.includes('Custom-Header value'));
  assert.ok(caddyfile.includes('complex.example.com'));
  assert.ok(caddyfile.includes('unknown_directive test'));
});

test('buildCaddyfileFromBlocks: snippet 保留块排在引用它的站点前', () => {
  const siteRaw = `example.com {
  import waf_protect
  reverse_proxy 127.0.0.1:9090
}`;
  const snippetRaw = `(waf_protect) {
  header X-Test preserved
}`;

  const draft: CaddyBlockDraft = {
    schemaVersion: 1,
    global: { raw: '# snippets' },
    upstreams: [],
    sites: [],
    preservedBlocks: [
      createPreservedBlock(siteRaw, '复杂站点', 'site'),
      createPreservedBlock(snippetRaw, '测试 snippet', 'snippet')
    ]
  };

  const caddyfile = buildCaddyfileFromBlocks(draft);

  assert.ok(caddyfile.includes('(waf_protect)'));
  assert.ok(caddyfile.includes('import waf_protect'));
  assert.ok(
    caddyfile.indexOf('(waf_protect)') < caddyfile.indexOf('import waf_protect'),
    'snippet 定义应排在 import 前'
  );
});

test('isEditableSite: 简单反代站点返回 true', () => {
  const site: Site = {
    id: 's1',
    name: 'test',
    enabled: true,
    domains: ['example.com'],
    tls: { mode: 'auto' },
    imports: [],
    geoip2Vars: [],
    encode: [],
    routes: [
      {
        id: 'r1',
        name: '默认路由',
        enabled: true,
        match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
        logAppend: [],
        handles: [
          {
            id: 'h1',
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
  assert.equal(isEditableSite(site), true);
});

test('isEditableSite: 包含 matcher 的站点返回 false', () => {
  const site: Site = {
    id: 's1',
    name: 'test',
    enabled: true,
    domains: ['example.com'],
    tls: { mode: 'auto' },
    imports: [],
    geoip2Vars: [],
    encode: [],
    routes: [
      {
        id: 'r1',
        name: '默认路由',
        enabled: true,
        match: { host: [], path: ['/api/*'], method: [], header: [], query: [], expression: '' },
        logAppend: [],
        handles: [
          {
            id: 'h1',
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
  assert.equal(isEditableSite(site), false);
});

test('原始配置保存路径不会携带旧 modules', () => {
  const config = `example.com {
  reverse_proxy 127.0.0.1:8080
}`;
  const draft = parseCaddyfileToBlocks(config);

  // 验证 draft 中 sites 的数据结构不包含旧 modules 字段
  assert.ok(!('modules' in draft));
  assert.ok(!('preservedBlocks' in draft.sites[0]));
  // preservedBlocks 只存在于 draft 顶层
  assert.ok(Array.isArray(draft.preservedBlocks));
});

test('createPreservedBlock 生成正确标题', () => {
  const block = createPreservedBlock(
    `example.com {
  reverse_proxy 127.0.0.1:8080
}`,
    '测试原因',
    'site'
  );

  assert.ok(block.id.length > 0);
  assert.equal(block.kind, 'site');
  assert.equal(block.reason, '测试原因');
  assert.ok(block.title.includes('example.com'));
});

test('parseCaddyfileToBlocks: 独立 snippet 在保存-重载后不丢失', () => {
  // 模拟一个典型的 Caddyfile：全局块 + 独立 snippet + 站点
  const config = `{
  order coraza_waf first
}

(common_headers) {
  header X-Frame-Options SAMEORIGIN
  header X-Content-Type-Options nosniff
}

example.com {
  reverse_proxy 127.0.0.1:8080
}`;

  const draft = parseCaddyfileToBlocks(config);

  // 站点应可编辑
  assert.equal(draft.sites.length, 1);
  assert.equal(draft.sites[0].domains[0], 'example.com');

  // 独立 snippet 必须在 preservedBlocks 中
  const snippetBlocks = draft.preservedBlocks.filter(b => b.kind === 'snippet');
  assert.ok(snippetBlocks.length >= 1, '独立 snippet 应被保留');
  assert.ok(snippetBlocks.some(b => b.raw.includes('common_headers')), 'preservedBlocks 应包含 common_headers');
  assert.ok(snippetBlocks.some(b => b.raw.includes('X-Frame-Options')), 'preservedBlocks 应包含 snippet 内容');

  // 构建 Caddyfile 后 snippet 内容不丢失
  const caddyfile = buildCaddyfileFromBlocks(draft);
  assert.ok(caddyfile.includes('common_headers'), '生成的 Caddyfile 应包含 snippet');
  assert.ok(caddyfile.includes('X-Frame-Options'), '生成的 Caddyfile 应包含 snippet 内容');
  assert.ok(caddyfile.includes('example.com'), '生成的 Caddyfile 应包含站点');
  assert.ok(caddyfile.includes('reverse_proxy'), '生成的 Caddyfile 应包含站点配置');
});
