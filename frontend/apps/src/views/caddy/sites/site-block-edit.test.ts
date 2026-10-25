import { describe, expect, it } from 'vitest';

import {
  applyBasicEditsToSiteBlock,
  parseManualTlsPaths,
  parseReverseProxyTarget,
  parseUpstreamSkipVerify,
  replaceSiteBlockInConfig,
} from './site-block-edit';

const SIMPLE = `app.example.com {
  tls internal
  handle {
    reverse_proxy localhost:3000 {
      lb_policy least_conn
      health_uri /health
    }
  }
}`;

const DNS_SITE = `pve.example.com {
  import cloudflare
  import waf_protect
  handle {
    reverse_proxy https://192.0.2.100:8006
  }
}`;

/** 无子块的 reverse_proxy（块形式收起的常见形态） */
const PLAIN = `plain.example.com {
  reverse_proxy localhost:4000
}`;

/** transport 是 reverse_proxy 唯一子内容的形态 */
const ONLY_TRANSPORT = `only.example.com {
  reverse_proxy localhost:5000 {
    transport http {
      tls_insecure_skip_verify
    }
  }
}`;

const MANUAL_SITE = `static.example.com {
  tls /config/certs/1/static.example.com/fullchain.pem /config/certs/1/static.example.com/privkey.pem
  import waf_protect
  handle {
    reverse_proxy localhost:9000
  }
}`;

const TLS_BLOCK_SITE = `block.example.com {
  tls {
    dns cloudflare {env.CF_API_TOKEN}
  }
  import waf_protect
  handle {
    reverse_proxy localhost:3000
  }
}`;

describe('applyBasicEditsToSiteBlock', () => {
  it('改域名：只动块头行', () => {
    const next = applyBasicEditsToSiteBlock(SIMPLE, { domains: ['new.example.com', 'alt.example.com'] });
    expect(next.split('\n')[0]).toBe('new.example.com alt.example.com {');
    expect(next).toContain('  tls internal');
    expect(next).toContain('      health_uri /health');
  });

  it('改上游：保留 reverse_proxy 子块与大括号', () => {
    const next = applyBasicEditsToSiteBlock(SIMPLE, { upstream: '10.0.0.5:8080' });
    expect(next).toContain('    reverse_proxy 10.0.0.5:8080 {');
    expect(next).toContain('      lb_policy least_conn');
    expect(next.split('\n')).toHaveLength(SIMPLE.split('\n').length);
  });

  it('TLS 切换：internal→dns 换行为 import cloudflare，auto 全清', () => {
    const dns = applyBasicEditsToSiteBlock(SIMPLE, { tlsMode: 'dns' });
    expect(dns).not.toContain('tls internal');
    expect(dns).toContain('  import cloudflare');

    const auto = applyBasicEditsToSiteBlock(DNS_SITE, { tlsMode: 'auto' });
    expect(auto).not.toContain('import cloudflare');
    expect(auto).toContain('import waf_protect');
    expect(auto).toContain('reverse_proxy https://192.0.2.100:8006');
  });

  it('TLS 切 off：插入 tls off 且不动能力 import', () => {
    const next = applyBasicEditsToSiteBlock(DNS_SITE, { tlsMode: 'off' });
    expect(next).toContain('  tls off');
    expect(next).not.toContain('import cloudflare');
    expect(next).toContain('import waf_protect');
  });

  it('多行 tls 块：整块删除不留学生行，dns 子行与闭括号一并清掉', () => {
    const next = applyBasicEditsToSiteBlock(TLS_BLOCK_SITE, { tlsMode: 'internal' });
    expect(next).toContain('  tls internal');
    expect(next).not.toContain('dns cloudflare');
    expect(next).not.toMatch(/^\s*\}\s*\n\s*\}/);
    expect(next).toContain('import waf_protect');
    expect(next).toContain('reverse_proxy localhost:3000');
    expect(next.split('\n')).toHaveLength(TLS_BLOCK_SITE.split('\n').length - 2);
  });

  it('多行 tls 块切 auto：整块删除后回退自动签发', () => {
    const next = applyBasicEditsToSiteBlock(TLS_BLOCK_SITE, { tlsMode: 'auto' });
    expect(next).not.toContain('tls');
    expect(next).not.toContain('dns cloudflare');
    expect(next).toContain('block.example.com {');
    expect(next).toContain('import waf_protect');
  });

  it('TLS 切 manual：改写为 tls <cert> <key>，既有单路径行被替换', () => {
    const next = applyBasicEditsToSiteBlock(SIMPLE, {
      tlsMode: 'manual',
      tlsCertPath: '/config/certs/1/app/fullchain.pem',
      tlsKeyPath: '/config/certs/1/app/privkey.pem',
    });
    expect(next).toContain(
      '  tls /config/certs/1/app/fullchain.pem /config/certs/1/app/privkey.pem',
    );
    expect(next).not.toContain('tls internal');
    expect(next).toContain('    reverse_proxy localhost:3000 {');
    expect(next).toContain('      health_uri /health');
  });

  it('manual 换路径：原地替换双路径行，其余字节不动', () => {
    const next = applyBasicEditsToSiteBlock(MANUAL_SITE, {
      tlsMode: 'manual',
      tlsCertPath: '/config/certs/2/static/fullchain.pem',
      tlsKeyPath: '/config/certs/2/static/privkey.pem',
    });
    expect(next).toContain(
      '  tls /config/certs/2/static/fullchain.pem /config/certs/2/static/privkey.pem',
    );
    expect(next).not.toContain('/config/certs/1/');
    expect(next).toContain('import waf_protect');
    expect(next).toContain('reverse_proxy localhost:9000');
    expect(next.split('\n')).toHaveLength(MANUAL_SITE.split('\n').length);
  });

  it('manual 路径不齐全：整个 TLS 编辑不动，不产生半行 tls', () => {
    expect(
      applyBasicEditsToSiteBlock(SIMPLE, { tlsMode: 'manual', tlsCertPath: '/c.pem' }),
    ).toBe(SIMPLE);
    expect(
      applyBasicEditsToSiteBlock(MANUAL_SITE, { tlsMode: 'manual' }),
    ).toBe(MANUAL_SITE);
  });

  it('manual 切 auto：双路径行删除，回退自动签发', () => {
    const next = applyBasicEditsToSiteBlock(MANUAL_SITE, { tlsMode: 'auto' });
    expect(next).not.toContain('tls ');
    expect(next).toContain('import waf_protect');
    expect(next).toContain('static.example.com {');
  });

  it('空编辑原样返回；无块头原样返回', () => {
    expect(applyBasicEditsToSiteBlock(SIMPLE, {})).toBe(SIMPLE);
    expect(applyBasicEditsToSiteBlock('plain', { upstream: 'x' })).toBe('plain');
  });

  it('CRLF 保持行尾', () => {
    const crlf = SIMPLE.split('\n').join('\r\n');
    const next = applyBasicEditsToSiteBlock(crlf, { domains: ['new.example.com'] });
    expect(next.split('\n')[0]).toBe('new.example.com {\r');
    expect(next).toContain('\r\n');
  });

  it('跳过验证开启：reverse_proxy 无子块时转为块形式并插入 transport', () => {
    const next = applyBasicEditsToSiteBlock(PLAIN, { upstreamSkipVerify: true });
    expect(next).toContain('  reverse_proxy localhost:4000 {');
    expect(next).toContain('    transport http {');
    expect(next).toContain('      tls_insecure_skip_verify');
    expect(next).toContain('    }');
    expect(next.trimEnd().endsWith('}')).toBe(true);
    expect(parseUpstreamSkipVerify(next)).toBe(true);
  });

  it('跳过验证开启：已有子块时插到块头之后，原子行整体后移', () => {
    const next = applyBasicEditsToSiteBlock(SIMPLE, { upstreamSkipVerify: true });
    expect(next).toContain('      transport http {');
    expect(next).toContain('        tls_insecure_skip_verify');
    expect(next).toContain('      lb_policy least_conn');
    expect(next).toContain('      health_uri /health');
    expect(parseUpstreamSkipVerify(next)).toBe(true);
  });

  it('跳过验证：重复开启字节级不变（幂等）', () => {
    const once = applyBasicEditsToSiteBlock(SIMPLE, { upstreamSkipVerify: true });
    const twice = applyBasicEditsToSiteBlock(once, { upstreamSkipVerify: true });
    expect(twice).toBe(once);
    expect(applyBasicEditsToSiteBlock(ONLY_TRANSPORT, { upstreamSkipVerify: true })).toBe(
      ONLY_TRANSPORT,
    );
  });

  it('跳过验证关闭：transport 是唯一子内容时整块收回并还原行形式', () => {
    const next = applyBasicEditsToSiteBlock(ONLY_TRANSPORT, { upstreamSkipVerify: false });
    expect(next).toBe(`only.example.com {
  reverse_proxy localhost:5000
}`);
    expect(parseUpstreamSkipVerify(next)).toBe(false);
  });

  it('跳过验证关闭：关闭后与原块一致（先开后关的往返）', () => {
    const opened = applyBasicEditsToSiteBlock(SIMPLE, { upstreamSkipVerify: true });
    const closed = applyBasicEditsToSiteBlock(opened, { upstreamSkipVerify: false });
    expect(closed).toBe(SIMPLE);
  });

  it('跳过验证关闭：transport 还有其他子行时只删 skip 行', () => {
    const withOther = `mix.example.com {
  reverse_proxy localhost:6000 {
    header_up X-Real-IP {remote_host}
    transport http {
      dial_timeout 5s
      tls_insecure_skip_verify
    }
  }
}`;
    const next = applyBasicEditsToSiteBlock(withOther, { upstreamSkipVerify: false });
    expect(next).not.toContain('tls_insecure_skip_verify');
    expect(next).toContain('transport http {');
    expect(next).toContain('dial_timeout 5s');
    expect(next).toContain('header_up X-Real-IP {remote_host}');
  });

  it('跳过验证：无 reverse_proxy 指令时原样返回', () => {
    const staticSite = `static.example.com {
  root * /srv
  file_server
}`;
    expect(applyBasicEditsToSiteBlock(staticSite, { upstreamSkipVerify: true })).toBe(
      staticSite,
    );
  });

  it('跳过验证：改上游与开启跳过验证一次生效', () => {
    const next = applyBasicEditsToSiteBlock(SIMPLE, {
      upstream: 'https://10.0.0.9:8443',
      upstreamSkipVerify: true,
    });
    expect(next).toContain('    reverse_proxy https://10.0.0.9:8443 {');
    expect(next).toContain('        tls_insecure_skip_verify');
    expect(next).toContain('      lb_policy least_conn');
  });

  it('跳过验证：CRLF 块内插入行保持 \\r\\n', () => {
    const crlf = PLAIN.split('\n').join('\r\n');
    const next = applyBasicEditsToSiteBlock(crlf, { upstreamSkipVerify: true });
    expect(next).toContain('tls_insecure_skip_verify\r\n');
    expect(next).toContain('transport http {\r\n');
  });
});

describe('replaceSiteBlockInConfig', () => {
  it('只替换首个出现的站点块，其余字节不动', () => {
    const config = `{
  admin localhost:2019
}

${SIMPLE}

other.example.com {
  reverse_proxy localhost:9
}
`;
    const newBlock = SIMPLE.replace('localhost:3000', 'localhost:4000');
    const next = replaceSiteBlockInConfig(config, SIMPLE, newBlock);
    expect(next).toContain('reverse_proxy localhost:4000');
    expect(next).toContain('other.example.com {');
    expect(next).toContain('admin localhost:2019');
    expect(next.split('other.example.com').length).toBe(2);
  });

  it('旧块不存在时原样返回', () => {
    expect(replaceSiteBlockInConfig('abc', 'xyz', 'zzz')).toBe('abc');
  });

  it('新块含 $&、$\'、$1 等替换模式串时按字面写入', () => {
    const newBlock = `rich.example.com {
  tls $& $' $1 $$
  reverse_proxy localhost:3000
}`;
    const next = replaceSiteBlockInConfig(`head\n${SIMPLE}\ntail`, SIMPLE, newBlock);
    expect(next).toContain(`tls $& $' $1 $$`);
    expect(next).toContain('head');
    expect(next).toContain('tail');
    // $& 被展开时会把旧块原文（tls internal）带进结果
    expect(next).not.toContain('tls internal');
  });
});

describe('parseManualTlsPaths', () => {
  it('识别双路径 tls 行', () => {
    expect(parseManualTlsPaths(MANUAL_SITE)).toEqual({
      certPath: '/config/certs/1/static.example.com/fullchain.pem',
      keyPath: '/config/certs/1/static.example.com/privkey.pem',
    });
  });

  it('单路径或无 tls 行返回 null', () => {
    expect(parseManualTlsPaths(SIMPLE)).toBeNull();
    expect(parseManualTlsPaths(DNS_SITE)).toBeNull();
    expect(parseManualTlsPaths('a.example.com {\n}')).toBeNull();
    expect(parseManualTlsPaths('')).toBeNull();
  });
});

describe('parseReverseProxyTarget', () => {
  it('解析块内首条 reverse_proxy 目标（含 https 前缀）', () => {
    expect(parseReverseProxyTarget(SIMPLE)).toBe('localhost:3000');
    expect(parseReverseProxyTarget(DNS_SITE)).toBe('https://192.0.2.100:8006');
    expect(parseReverseProxyTarget(ONLY_TRANSPORT)).toBe('localhost:5000');
  });

  it('无 reverse_proxy 或块形式占位返回 null', () => {
    expect(parseReverseProxyTarget('a.example.com {\n  file_server\n}')).toBeNull();
    expect(parseReverseProxyTarget('a.example.com {\n  reverse_proxy {\n  }\n}')).toBeNull();
    expect(parseReverseProxyTarget('')).toBeNull();
  });
});

describe('parseUpstreamSkipVerify', () => {
  it('transport http 内有 tls_insecure_skip_verify 判开启', () => {
    expect(parseUpstreamSkipVerify(ONLY_TRANSPORT)).toBe(true);
    const opened = applyBasicEditsToSiteBlock(SIMPLE, { upstreamSkipVerify: true });
    expect(parseUpstreamSkipVerify(opened)).toBe(true);
  });

  it('无 transport 或无 skip 行判关闭', () => {
    expect(parseUpstreamSkipVerify(SIMPLE)).toBe(false);
    expect(parseUpstreamSkipVerify(PLAIN)).toBe(false);
    expect(parseUpstreamSkipVerify('a.example.com {\n  file_server\n}')).toBe(false);
  });
});
