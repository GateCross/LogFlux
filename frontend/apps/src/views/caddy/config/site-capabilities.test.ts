import { describe, expect, it } from 'vitest';

import {
  applyCapabilityToSiteBlock,
  applySiteLogToSiteBlock,
  applyWafModeToSiteBlock,
  describeSiteCapabilities,
  extractSiteCapabilities,
  renderCapabilityLines,
  WAF_DETECT_IMPORT,
} from './site-capabilities';
import type { SiteLogSettings } from './site-log-settings';
import { DEFAULT_SITE_LOG_SETTINGS } from './site-log-settings';

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

  it('徽章描述覆盖全部能力', () => {
    const badges = describeSiteCapabilities(extractSiteCapabilities(PVE_SITE));
    expect(badges.map(item => item.label)).toEqual([
      '维护',
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

  it('维护模式开启、重复开启幂等与完全还原', () => {
    const onLines = renderCapabilityLines('maintenance', { enabled: true });
    const withMaint = applyCapabilityToSiteBlock(PVE_SITE, 'maintenance', onLines);
    expect(withMaint).toContain('@maintenance path *');
    expect(withMaint).toContain('respond "站点维护中，请稍后再试。" 503');
    const caps = extractSiteCapabilities(withMaint);
    expect(caps.maintenance.status).toBe('on');

    // 重复开启幂等（字节级不变）
    const twice = applyCapabilityToSiteBlock(withMaint, 'maintenance', onLines);
    expect(twice).toBe(withMaint);

    // 关闭维护模式：必须完全还原为原始字节
    const restored = applyCapabilityToSiteBlock(withMaint, 'maintenance', []);
    expect(restored).not.toContain('@maintenance');
    expect(restored).not.toContain('503');
    expect(extractSiteCapabilities(restored).maintenance.status).toBe('off');
    expect(restored).toBe(PVE_SITE);
  });

  it('无块头的文本原样返回', () => {
    expect(applyCapabilityToSiteBlock('just text', 'waf', ['import waf_protect'])).toBe('just text');
  });
});

const BLOCK_LOG_SITE = `myapp.example.com {
  import subdomain-log {
    name myapp
    size 20
    level ERROR
  }
  handle {
    reverse_proxy localhost:3000
  }
}`;

const TAB_SITE = `tab.example.com {
\timport waf_protect
\thandle {
\t\treverse_proxy localhost:3000
\t}
}`;

function makeLog(overrides: Partial<SiteLogSettings> = {}): SiteLogSettings {
  return { ...DEFAULT_SITE_LOG_SETTINGS, ...overrides };
}

describe('accessLog 能力识别兼容块参数写法', () => {
  it('带块参数的 import subdomain-log 识别为开启', () => {
    expect(extractSiteCapabilities(BLOCK_LOG_SITE).accessLog.status).toBe('on');
  });

  it('旧单行写法仍识别为开启', () => {
    expect(extractSiteCapabilities(LAN_SITE).accessLog.status).toBe('on');
  });

  it('块参数 import 不被误计入 otherImports', () => {
    expect(extractSiteCapabilities(BLOCK_LOG_SITE).otherImports).toEqual([]);
  });
});

describe('applySiteLogToSiteBlock', () => {
  it('裸站点开日志：块头后插入位置参数单行，其余字节不动', () => {
    expect(applySiteLogToSiteBlock(BARE_SITE, makeLog(), 'plain')).toBe(
      `plain.example.com {
  import subdomain-log plain 10 INFO
  handle {
    reverse_proxy localhost:3000
  }
}`,
    );
  });

  it('名字缺省时回落到块头域名前缀', () => {
    const next = applySiteLogToSiteBlock(BARE_SITE, makeLog());
    expect(next).toContain('import subdomain-log plain 10 INFO');
  });

  it('仅记错误：level 写 ERROR', () => {
    const next = applySiteLogToSiteBlock(BARE_SITE, makeLog({ access: false }), 'plain');
    expect(next).toContain('import subdomain-log plain 10 ERROR');
  });

  it('两开关全关：清干净且不写任何行', () => {
    const next = applySiteLogToSiteBlock(
      BLOCK_LOG_SITE,
      makeLog({ access: false, error: false }),
    );
    expect(next).not.toContain('subdomain-log');
    // 整块 5 行（import 行 + 三个参数 + 闭括号）全部消失
    expect(next.split('\n')).toHaveLength(BLOCK_LOG_SITE.split('\n').length - 5);
    expect(next).toContain('    reverse_proxy localhost:3000');
  });

  it('旧的单行写法整行被新写法替换，不留下两份配置', () => {
    const next = applySiteLogToSiteBlock(LAN_SITE, makeLog(), 'local');
    expect(next.match(/import subdomain-log/g)).toHaveLength(1);
    expect(next).not.toContain('import subdomain-log local\n');
    expect(next).toContain('import subdomain-log local 10 INFO');
  });

  it('既有块写法只改设置：文件名沿用原参数', () => {
    const next = applySiteLogToSiteBlock(BLOCK_LOG_SITE, makeLog({ sizeMb: 32 }));
    expect(next).toContain('import subdomain-log myapp 32 INFO');
    expect(next.match(/subdomain-log/g)).toHaveLength(1);
  });

  it('幂等：连续应用两次结果一致', () => {
    const once = applySiteLogToSiteBlock(BARE_SITE, makeLog(), 'plain');
    expect(applySiteLogToSiteBlock(once, makeLog(), 'plain')).toBe(once);
  });

  it('内容一致的重复应用返回原文本（字节级不变）', () => {
    const once = applySiteLogToSiteBlock(BARE_SITE, makeLog(), 'plain');
    const twice = applySiteLogToSiteBlock(once, makeLog(), 'plain');
    expect(twice).toBe(once);
  });

  it('单行写法插入后仍是 CRLF', () => {
    const crlf = BARE_SITE.split('\n').join('\r\n');
    const next = applySiteLogToSiteBlock(crlf, makeLog(), 'plain');
    expect(next.split('\r\n')[1]).toBe('  import subdomain-log plain 10 INFO');
    expect(next.includes('\r\n')).toBe(true);
  });

  it('tab 缩进站点沿用 Tab，不混入空格缩进', () => {
    const next = applySiteLogToSiteBlock(TAB_SITE, makeLog(), 'tab');
    expect(next).toContain('\timport subdomain-log tab 10 INFO\n');
    expect(next).not.toContain('  import subdomain-log');
  });

  it('保留块内注释与其它指令顺序', () => {
    const withComment = `a.example.com {
  # 保留这行注释
  import waf_protect
  handle {
    reverse_proxy localhost:3000
  }
}`;
    const next = applySiteLogToSiteBlock(withComment, makeLog(), 'a');
    expect(next).toContain('  # 保留这行注释\n');
    expect(next.indexOf('# 保留这行注释')).toBeLessThan(
      next.indexOf('import waf_protect'),
    );
  });

  it('无块头或空文本原样返回', () => {
    expect(applySiteLogToSiteBlock('just text', makeLog(), 'x')).toBe('just text');
    expect(applySiteLogToSiteBlock('', makeLog(), 'x')).toBe('');
  });
});

describe('站点级 WAF 引擎模式', () => {
  it('挂哪个 snippet 就是哪个模式', () => {
    expect(extractSiteCapabilities(PVE_SITE).waf.mode).toBe('protect');
    const detectSite = PVE_SITE.replace('import waf_protect', `import ${WAF_DETECT_IMPORT}`);
    expect(extractSiteCapabilities(detectSite).waf.mode).toBe('detect');
    const bare = 'a.example.com {\n  reverse_proxy localhost:3000\n}';
    expect(extractSiteCapabilities(bare).waf.mode).toBe('off');
  });

  it('内联 coraza_waf 认不出模式时判 unknown，不猜测', () => {
    const inline = `a.example.com {\n  coraza_waf {\n    directives \`\n    SecRuleEngine On\n    \`\n  }\n}`;
    expect(extractSiteCapabilities(inline).waf.mode).toBe('unknown');
  });

  it('切模式只替换那一行 import，其余字节不动', () => {
    const detect = applyWafModeToSiteBlock(PVE_SITE, 'detect');
    expect(detect).toContain(`  import ${WAF_DETECT_IMPORT}`);
    expect(detect).not.toContain('import waf_protect\n');
    // 其它能力行、缩进、块体一律保留
    for (const keep of ['import deny_403_redirect', 'import cloudflare', 'import geo_auth', 'tls_insecure_skip_verify']) {
      expect(detect).toContain(keep);
    }

    // 切回拦截正好还原原文
    expect(applyWafModeToSiteBlock(detect, 'protect')).toBe(PVE_SITE);

    // 关闭：两种挂载都不留
    const off = applyWafModeToSiteBlock(PVE_SITE, 'off');
    expect(off).not.toContain('waf_protect');
    expect(extractSiteCapabilities(off).waf.mode).toBe('off');
  });

  it('连续应用幂等', () => {
    const once = applyWafModeToSiteBlock(PVE_SITE, 'detect');
    expect(applyWafModeToSiteBlock(once, 'detect')).toBe(once);
  });

  it('一个站点的模式改写不影响相邻站点', () => {
    const siteA = 'a.example.com {\n  import waf_protect\n}';
    const siteB = 'b.example.com {\n  import waf_protect\n}';
    const changedA = applyWafModeToSiteBlock(siteA, 'detect');
    // 两段文本独立处理：B 的字节完全不变
    expect(applyWafModeToSiteBlock(siteB, 'protect')).toBe(siteB);
    expect(changedA).not.toBe(siteA);
  });
});
