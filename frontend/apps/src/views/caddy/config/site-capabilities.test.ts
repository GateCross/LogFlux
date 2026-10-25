import { describe, expect, it } from 'vitest';

import {
  applyCapabilityToSiteBlock,
  describeSiteCapabilities,
  extractSiteCapabilities,
  renderCapabilityLines,
} from './site-capabilities';

const PVE_SITE = `pve.example.com {
  import deny_403_redirect
  import waf_protect
  import cloudflare
  import geo_auth
  handle {
    reverse_proxy https://192.0.2.100:8006 {
      transport http {
        tls_insecure_skip_verify
      }
    }
  }
}`;

const LAN_SITE = `:80 {
  import waf_protect
  import compression
  import subdomain-log local

  @lan remote_ip 127.0.0.1/8 10.0.0.0/8
  route @lan {
    @ikuai host ikuai.example.com
    handle @ikuai {
      reverse_proxy https://192.0.2.254:443
    }
    handle {
      import health_check
      import geo_auth
      import api_proxy
      import frontend_simple
    }
  }
  route {
    redir https://{host}{uri} permanent
  }
}`;

const BARE_SITE = `plain.example.com {
  handle {
    reverse_proxy localhost:3000
  }
}`;

const INLINE_WAF_SITE = `custom.example.com {
  coraza_waf {
    directives \`
    SecRuleEngine On
    \`
  }
  handle {
    reverse_proxy localhost:3000
  }
}`;

const INLINE_ENCODE_SITE = `encoded.example.com {
  encode zstd gzip
  handle {
    reverse_proxy localhost:3000
  }
}`;

const INLINE_TLS_SITE = `tls-site.example.com {
  tls /certs/a.pem /certs/a.key
  handle {
    reverse_proxy localhost:3000
  }
}`;

describe('extractSiteCapabilities', () => {
  it('识别线上 pve 站点的全部能力', () => {
    const caps = extractSiteCapabilities(PVE_SITE);
    expect(caps.waf.status).toBe('on');
    expect(caps.geo.status).toBe('on');
    expect(caps.errorPage.status).toBe('on');
    expect(caps.compression.status).toBe('off');
    expect(caps.tls.mode).toBe('dns');
    expect(caps.tls.status).toBe('on');
    expect(caps.otherImports).toEqual([]);
  });

  it('识别 :80 内网站点：嵌套 import 也算开启，业务 import 归入 other', () => {
    const caps = extractSiteCapabilities(LAN_SITE);
    expect(caps.waf.status).toBe('on');
    expect(caps.compression.status).toBe('on');
    expect(caps.accessLog.status).toBe('on');
    expect(caps.geo.status).toBe('on');
    expect(caps.otherImports).toEqual(['health_check', 'api_proxy', 'frontend_simple']);
  });

  it('裸站点全部能力为 off，TLS 走 auto', () => {
    const caps = extractSiteCapabilities(BARE_SITE);
    expect(caps.waf.status).toBe('off');
    expect(caps.geo.status).toBe('off');
    expect(caps.tls.mode).toBe('auto');
    expect(caps.tls.status).toBe('off');
    expect(caps.otherImports).toEqual([]);
  });

  it('内联 coraza_waf 判 unknown，不误报为 off', () => {
    const caps = extractSiteCapabilities(INLINE_WAF_SITE);
    expect(caps.waf.status).toBe('unknown');
    expect(caps.waf.note).toContain('原文模式');
  });

  it('内联 encode 识别为压缩开启', () => {
    const caps = extractSiteCapabilities(INLINE_ENCODE_SITE);
    expect(caps.compression.status).toBe('on');
  });

  it('手动证书路径识别为 manual', () => {
    const caps = extractSiteCapabilities(INLINE_TLS_SITE);
    expect(caps.tls.mode).toBe('manual');
    expect(caps.tls.status).toBe('on');
  });

  it('徽章描述覆盖六个能力', () => {
    const badges = describeSiteCapabilities(extractSiteCapabilities(PVE_SITE));
    expect(badges.map(item => item.label)).toEqual([
      'WAF',
      'IP 限制',
      '安全头',
      '错误页',
      '压缩',
      '日志',
    ]);
    expect(badges.filter(item => item.status === 'on').map(item => item.label)).toEqual([
      'WAF',
      'IP 限制',
      '错误页',
    ]);
  });
});

describe('applyCapabilityToSiteBlock', () => {
  it('给裸站点开启 WAF：只在块头后插一行，其余字节不动', () => {
    const next = applyCapabilityToSiteBlock(
      BARE_SITE,
      'waf',
      renderCapabilityLines('waf', { enabled: true }),
    );
    expect(next).toBe(`plain.example.com {
  import waf_protect
  handle {
    reverse_proxy localhost:3000
  }
}`);
  });

  it('关闭 WAF：恰好少一行 import waf_protect', () => {
    const next = applyCapabilityToSiteBlock(PVE_SITE, 'waf', []);
    expect(next).not.toContain('import waf_protect');
    expect(next.split('\n')).toHaveLength(PVE_SITE.split('\n').length - 1);
    expect(next).toContain('import deny_403_redirect');
    expect(next).toContain('import geo_auth');
  });

  it('幂等：连续应用两次结果一致', () => {
    const once = applyCapabilityToSiteBlock(
      BARE_SITE,
      'waf',
      renderCapabilityLines('waf', { enabled: true }),
    );
    const twice = applyCapabilityToSiteBlock(
      once,
      'waf',
      renderCapabilityLines('waf', { enabled: true }),
    );
    expect(twice).toBe(once);
  });

  it('已开启的站点重复开启不产生重复行', () => {
    const next = applyCapabilityToSiteBlock(
      PVE_SITE,
      'waf',
      renderCapabilityLines('waf', { enabled: true }),
    );
    expect(next.match(/import waf_protect/g)).toHaveLength(1);
    expect(next).toBe(PVE_SITE);
  });

  it('日志能力带参数渲染并在再次应用时原位替换', () => {
    const withLog = applyCapabilityToSiteBlock(
      BARE_SITE,
      'accessLog',
      renderCapabilityLines('accessLog', { enabled: true, name: 'myapp' }),
    );
    expect(withLog).toContain('import subdomain-log myapp');
    const renamed = applyCapabilityToSiteBlock(
      withLog,
      'accessLog',
      renderCapabilityLines('accessLog', { enabled: true, name: 'other' }),
    );
    expect(renamed).toContain('import subdomain-log other');
    expect(renamed).not.toContain('import subdomain-log myapp');
    expect(renamed.match(/import subdomain-log/g)).toHaveLength(1);
  });

  it('关闭压缩时同时移除 snippet import 与内联 encode 行', () => {
    const next = applyCapabilityToSiteBlock(INLINE_ENCODE_SITE, 'compression', []);
    expect(next).not.toMatch(/^[ \t]*encode\s/m);
    expect(next).toContain('reverse_proxy localhost:3000');
  });

  it('CRLF 文本保持 CRLF', () => {
    const crlf = BARE_SITE.split('\n').join('\r\n');
    const next = applyCapabilityToSiteBlock(
      crlf,
      'waf',
      renderCapabilityLines('waf', { enabled: true }),
    );
    expect(next).toContain('\r\n');
    expect(next.split('\r\n')[1]).toBe('  import waf_protect');
  });

  it('无块头的文本原样返回', () => {
    expect(applyCapabilityToSiteBlock('just text', 'waf', ['import waf_protect'])).toBe('just text');
  });
});
