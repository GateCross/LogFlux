import { describe, expect, it } from 'vitest';

import { buildCaddyfileFromBlocks, parseCaddyfileToBlocks } from './caddy-config-blocks';

/**
 * 回归：块前置注释的归属。
 *
 * 曾经的问题：站点块前的注释被归到 global.raw，重建时被提到文件顶部；
 * 修复后注释要么随块保留，要么随块丢弃，但绝不能跑到 global 里。
 */
describe('块前置注释归属', () => {
  const config = `{
  admin localhost:2019
}

# 属于 ikuai 的说明
ikuai.example.com {
  reverse_proxy http://10.0.0.1
}

# 属于 pve 的说明
pve.example.com {
  reverse_proxy https://10.0.0.2:8006
}
`;

  it('站点前置注释不得进入 global.raw', () => {
    const draft = parseCaddyfileToBlocks(config);
    expect(draft.global?.raw ?? '').not.toContain('属于 ikuai 的说明');
    expect(draft.global?.raw ?? '').not.toContain('属于 pve 的说明');
  });

  it('重建后不丢站点块', () => {
    const draft = parseCaddyfileToBlocks(config);
    const rebuilt = buildCaddyfileFromBlocks(draft, { sourceOrder: config });
    expect(rebuilt).toContain('ikuai.example.com');
    expect(rebuilt).toContain('pve.example.com');
    expect(rebuilt).toContain('admin localhost:2019');
  });
});

/**
 * 回归：解析 → 重建必须无损。
 *
 * 生产配置含大量 snippet 与 route，站点会被判定为复杂块原样保留；
 * 这里锁定「原文每一行都还在」这个底线，避免解析器悄悄吞掉指令。
 */
describe('解析重建无损', () => {
  const config = `{
  admin localhost:2019
  auto_https disable_redirects
  order coraza_waf first
}

(geo_auth) {
  forward_auth localhost:8888 {
    uri /api/internal/geo-check
  }
}

(waf_protect) {
  coraza_waf {
    load_owasp_crs
    directives \`
    SecAction "id:900000,setvar:tx.blocking_paranoia_level=2"
    Include @owasp_crs/*.conf
    \`
  }
}

:80 {
  import waf_protect

  @lan remote_ip 192.168.0.0/16
  route @lan {
    import geo_auth
    respond "lan" 200
  }
}

site.example.com {
  import waf_protect
  reverse_proxy https://192.0.2.254:443 {
    transport http {
      tls_insecure_skip_verify
    }
  }
}
`;

  it('原文每一行都保留', () => {
    const draft = parseCaddyfileToBlocks(config);
    const rebuilt = buildCaddyfileFromBlocks(draft, { sourceOrder: config });

    const rebuiltLines = new Set(
      rebuilt.split('\n').map((line) => line.trim()).filter(Boolean),
    );
    const lost = config
      .split('\n')
      .map((line) => line.trim())
      .filter((line) => line && !rebuiltLines.has(line));

    expect(lost).toEqual([]);
  });
});
