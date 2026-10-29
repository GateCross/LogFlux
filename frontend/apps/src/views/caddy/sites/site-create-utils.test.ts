import { describe, expect, it } from 'vitest';

import {
  appendSiteBlockToConfig,
  buildCreatedSiteBlock,
  createDefaultSiteForm,
  findDuplicateDomains,
  isValidDomainOrListen,
  validateCreateSiteForm,
} from './site-create-utils';

const EXISTING = `{
  admin localhost:2019
}

(cloudflare) {
  tls {
    dns cloudflare TOKEN
  }
}

app.example.com {
  reverse_proxy localhost:3000
}
`;

describe('isValidDomainOrListen', () => {
  it.each(['a.com', '*.a.com', ':80', 'localhost:8080', '192.168.1.10', 'pve.example.com:8006'])(
    '接受 %s',
    (value) => {
      expect(isValidDomainOrListen(value)).toBe(true);
    },
  );
  it.each(['', 'a..com', 'a b', 'example.com:abc', 'https://a.com'])('拒绝 %s', (value) => {
    expect(isValidDomainOrListen(value)).toBe(false);
  });
});

describe('validateCreateSiteForm', () => {
  it('完整表单通过', () => {
    const form = { ...createDefaultSiteForm(), domains: ['new.example.com'] };
    expect(validateCreateSiteForm(form, EXISTING)).toEqual([]);
  });

  it('上游带 path 被拦截（Caddy 上游不支持路径）', () => {
    const form = {
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
      upstream: 'https://192.168.50.10:10000/myserver',
    };
    expect(
      validateCreateSiteForm(form, EXISTING).some((item) => item.includes('不支持路径')),
    ).toBe(true);
  });

  it('域名重复被拦截', () => {
    const form = { ...createDefaultSiteForm(), domains: ['app.example.com'] };
    const errors = validateCreateSiteForm(form, EXISTING);
    expect(errors.some((item) => item.includes('app.example.com'))).toBe(true);
  });

  it('DNS 签发要求 cloudflare 片段存在', () => {
    const form = {
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
      tlsMode: 'dns' as const,
    };
    expect(validateCreateSiteForm(form, EXISTING)).toEqual([]);
    expect(
      validateCreateSiteForm(form, 'app.example.com {\n  reverse_proxy x\n}\n').some((item) =>
        item.includes('cloudflare'),
      ),
    ).toBe(true);
  });

  it('健康检查路径与时长格式校验', () => {
    const form = {
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
      healthEnabled: true,
      healthPath: 'health',
      healthInterval: '10x',
    };
    const errors = validateCreateSiteForm(form, EXISTING);
    expect(errors.some((item) => item.includes('以 / 开头'))).toBe(true);
    expect(errors.some((item) => item.includes('间隔'))).toBe(true);
  });

  it('已有证书模式未选择证书时拦截', () => {
    const form = {
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
      tlsMode: 'manual' as const,
    };
    expect(
      validateCreateSiteForm(form, EXISTING).some((item) => item.includes('已有证书')),
    ).toBe(true);
  });

  it('已有证书模式选择证书后通过', () => {
    const form = {
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
      tlsMode: 'manual' as const,
      manualCertPath: '/config/certs/1/a.example.com/fullchain.pem',
      manualKeyPath: '/config/certs/1/a.example.com/privkey.pem',
    };
    expect(validateCreateSiteForm(form, EXISTING)).toEqual([]);
  });
});

describe('findDuplicateDomains', () => {
  it('跨块头识别既有域名', () => {
    expect(findDuplicateDomains(EXISTING, ['app.example.com', 'fresh.io'])).toEqual([
      'app.example.com',
    ]);
  });
});

describe('buildCreatedSiteBlock', () => {
  it('默认表单生成最小反代块', () => {
    const block = buildCreatedSiteBlock({
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
    });
    expect(block).toBe(`new.example.com {
  handle {
    reverse_proxy localhost:8080
  }
}`);
  });

  it('多域名与 TLS 选项', () => {
    const block = buildCreatedSiteBlock({
      ...createDefaultSiteForm(),
      domains: ['a.example.com', 'b.example.com'],
      tlsMode: 'dns',
    });
    expect(block.startsWith('a.example.com b.example.com {')).toBe(true);
    expect(block).toContain('  import cloudflare');
  });

  it('已有证书（manual）按 tls 双路径写入', () => {
    const block = buildCreatedSiteBlock({
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
      tlsMode: 'manual',
      manualCertPath: '/config/certs/1/a.example.com/fullchain.pem',
      manualKeyPath: '/config/certs/1/a.example.com/privkey.pem',
    });
    expect(block).toContain(
      '  tls /config/certs/1/a.example.com/fullchain.pem /config/certs/1/a.example.com/privkey.pem',
    );
  });

  it('manual 路径残缺时不渲染 tls 行（防半行指令，交由校验拦截）', () => {
    const block = buildCreatedSiteBlock({
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
      tlsMode: 'manual',
      manualCertPath: '/config/certs/1/a.example.com/fullchain.pem',
    });
    expect(block).not.toMatch(/^[ \t]*tls /m);
  });

  it('负载策略与健康检查进 reverse_proxy 子块', () => {
    const block = buildCreatedSiteBlock({
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
      lbPolicy: 'least_conn',
      healthEnabled: true,
    });
    expect(block).toContain(`    reverse_proxy localhost:8080 {`);
    expect(block).toContain('      lb_policy least_conn');
    expect(block).toContain('      health_uri /health');
    expect(block).toContain('      health_interval 10s');
    expect(block).toContain('      health_timeout 5s');
  });
});

describe('appendSiteBlockToConfig', () => {
  it('中间恰好一个空行，收尾一个换行', () => {
    const block = buildCreatedSiteBlock({
      ...createDefaultSiteForm(),
      domains: ['new.example.com'],
    });
    const next = appendSiteBlockToConfig(`${EXISTING}\n\n\n`, block);
    expect(next).toBe(`${EXISTING.trimEnd()}\n\n${block}\n`);
    // 其余行字节级不变
    expect(next.startsWith(EXISTING.trimEnd())).toBe(true);
  });

  it('空配置直接返回新块', () => {
    const block = buildCreatedSiteBlock({
      ...createDefaultSiteForm(),
      domains: [':8080'],
    });
    expect(appendSiteBlockToConfig('   \n', block)).toBe(`${block}\n`);
  });
});
