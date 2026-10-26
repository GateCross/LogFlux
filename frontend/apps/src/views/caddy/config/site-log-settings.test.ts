import { describe, expect, it } from 'vitest';

import type { SiteLogSettings } from './site-log-settings';
import {
  DEFAULT_SITE_LOG_SETTINGS,
  levelToSettings,
  normalizeSiteLogSizeMb,
  parseSiteLogImport,
  renderLogSnippet,
  renderSiteImportBlock,
  SITE_LOG_FORCED_ERROR_MESSAGE,
  settingsToLevel,
  validateSiteLogSettings,
} from './site-log-settings';

function make(overrides: Partial<SiteLogSettings> = {}): SiteLogSettings {
  return { ...DEFAULT_SITE_LOG_SETTINGS, ...overrides };
}

describe('settingsToLevel 四象限', () => {
  it('开访问 + 开错误 → INFO', () => {
    expect(settingsToLevel(make({ access: true, error: true }))).toBe('INFO');
  });

  it('关访问 + 开错误 → ERROR', () => {
    expect(settingsToLevel(make({ access: false, error: true }))).toBe('ERROR');
  });

  it('关访问 + 关错误 → OFF', () => {
    expect(settingsToLevel(make({ access: false, error: false }))).toBe('OFF');
  });

  it('开访问 + 关错误 → INFO（level 是最低级别，该组合无法表达）', () => {
    expect(settingsToLevel(make({ access: true, error: false }))).toBe('INFO');
  });
});

describe('levelToSettings 反向映射', () => {
  it('info 还原为全开', () => {
    expect(levelToSettings('INFO', 10)).toEqual({
      access: true,
      error: true,
      sizeMb: 10,
    });
  });

  it('error 还原为只记错误', () => {
    expect(levelToSettings('ERROR', 20)).toEqual({
      access: false,
      error: true,
      sizeMb: 20,
    });
  });

  it('off 还原为全关', () => {
    expect(levelToSettings('OFF', 5)).toEqual({
      access: false,
      error: false,
      sizeMb: 5,
    });
  });

  it('大小写与空白不敏感', () => {
    expect(levelToSettings('error', 3)).toEqual(levelToSettings('  ERROR ', 3));
    expect(levelToSettings('Info', 3).access).toBe(true);
    expect(levelToSettings('off', 3).error).toBe(false);
  });

  it('未知值按全开兜底', () => {
    expect(levelToSettings('DEBUG', 10)).toEqual(make({ sizeMb: 10 }));
    expect(levelToSettings('', 10)).toEqual(make({ sizeMb: 10 }));
    expect(levelToSettings('WARN', 10)).toEqual(make({ sizeMb: 10 }));
  });

  it('与 settingsToLevel 在可表达组合上互逆', () => {
    for (const settings of [
      make({ access: true, error: true }),
      make({ access: false, error: true }),
      make({ access: false, error: false }),
    ]) {
      expect(levelToSettings(settingsToLevel(settings), settings.sizeMb)).toEqual(
        settings,
      );
    }
  });
});

describe('sizeMb clamp', () => {
  it('下界 1MB、上界 1024MB', () => {
    expect(normalizeSiteLogSizeMb(0)).toBe(1);
    expect(normalizeSiteLogSizeMb(-8)).toBe(1);
    expect(normalizeSiteLogSizeMb(1024)).toBe(1024);
    expect(normalizeSiteLogSizeMb(5000)).toBe(1024);
  });

  it('取整后 clamp，非数值回落默认 10MB', () => {
    expect(normalizeSiteLogSizeMb(10.6)).toBe(11);
    expect(normalizeSiteLogSizeMb(0.4)).toBe(1);
    expect(normalizeSiteLogSizeMb(Number.NaN)).toBe(10);
    expect(normalizeSiteLogSizeMb(Number.POSITIVE_INFINITY)).toBe(10);
  });

  it('渲染时大小上限同样被 clamp', () => {
    expect(renderSiteImportBlock(make({ sizeMb: 0 }), 'a')).toContain('  size 1');
    expect(renderSiteImportBlock(make({ sizeMb: 99999 }), 'a')).toContain(
      '  size 1024',
    );
    expect(renderSiteImportBlock(make({ sizeMb: 25.4 }), 'a')).toContain(
      '  size 26',
    );
  });
});

describe('renderLogSnippet 逐字比对', () => {
  it('渲染参数化 snippet 定义，2 空格缩进、LF 换行', () => {
    expect(renderLogSnippet(DEFAULT_SITE_LOG_SETTINGS)).toBe(
      [
        '(subdomain-log) {',
        '  log {',
        '    output file /var/log/caddy/{args.name}.log {',
        '      roll_size {args.size}mb',
        '      roll_keep 1',
        '    }',
        '    level {args.level}',
        '    format json',
        '  }',
        '}',
      ].join('\n'),
    );
  });

  it('定义与站点参数无关：任何设置下输出一致', () => {
    const baseline = renderLogSnippet(DEFAULT_SITE_LOG_SETTINGS);
    expect(renderLogSnippet(make({ access: false, error: true, sizeMb: 1 }))).toBe(
      baseline,
    );
    expect(renderLogSnippet(make({ access: false, error: false }))).toBe(baseline);
  });

  it('不含 CRLF 与 Tab', () => {
    const snippet = renderLogSnippet(DEFAULT_SITE_LOG_SETTINGS);
    expect(snippet).not.toContain('\r');
    expect(snippet).not.toContain('\t');
  });
});

describe('renderSiteImportBlock', () => {
  it('全开时渲染两行参数块（不带缩进）', () => {
    expect(renderSiteImportBlock(make(), 'myapp')).toEqual([
      'import subdomain-log {',
      '  name myapp',
      '  size 10',
      '  level INFO',
      '}',
    ]);
  });

  it('只记错误时 level 为 ERROR', () => {
    expect(renderSiteImportBlock(make({ access: false }), 'myapp')).toEqual([
      'import subdomain-log {',
      '  name myapp',
      '  size 10',
      '  level ERROR',
      '}',
    ]);
  });

  it('两开关全关时返回空数组，不写非法 level OFF', () => {
    expect(renderSiteImportBlock(make({ access: false, error: false }), 'myapp'))
      .toEqual([]);
  });

  it('文件名取首个 token，空名回落 site', () => {
    expect(renderSiteImportBlock(make(), '  my app  ')[1]).toBe('  name my');
    expect(renderSiteImportBlock(make(), '')[1]).toBe('  name site');
    expect(renderSiteImportBlock(make(), '   ')[1]).toBe('  name site');
  });
});

describe('parseSiteLogImport', () => {
  const BLOCK_SITE = `myapp.example.com {
  import subdomain-log {
    name myapp
    size 20
    level ERROR
  }
  handle {
    reverse_proxy localhost:3000
  }
}`;

  const LEGACY_SITE = `legacy.example.com {
  import subdomain-log legacy
  handle {
    reverse_proxy localhost:3000
  }
}`;

  it('解析块写法：名字与设置全部读回', () => {
    expect(parseSiteLogImport(BLOCK_SITE)).toEqual({
      name: 'myapp',
      settings: { access: false, error: true, sizeMb: 20 },
    });
  });

  it('解析旧单行写法：名字读回、设置用默认值', () => {
    expect(parseSiteLogImport(LEGACY_SITE)).toEqual({
      name: 'legacy',
      settings: { ...DEFAULT_SITE_LOG_SETTINGS },
    });
  });

  it('参数写在括号同一行也能解析', () => {
    const inline = `a.example.com {
  import subdomain-log { name a size 32 level INFO }
}`;
    expect(parseSiteLogImport(inline)).toEqual({
      name: 'a',
      settings: { access: true, error: true, sizeMb: 32 },
    });
  });

  it('块内整行注释不影响键值配对', () => {
    const commented = `a.example.com {
  import subdomain-log {
    # 日志文件名
    name a
    size 8
    level INFO
  }
}`;
    expect(parseSiteLogImport(commented)).toEqual({
      name: 'a',
      settings: { access: true, error: true, sizeMb: 8 },
    });
  });

  it('crlf 文本同样可解析', () => {
    expect(parseSiteLogImport(BLOCK_SITE.replace(/\n/g, '\r\n'))).toEqual(
      parseSiteLogImport(BLOCK_SITE),
    );
  });

  it('没有 import 时返回 null，不误配同前缀的其它 snippet', () => {
    expect(
      parseSiteLogImport('a.example.com {\n  import subdomain-logger x\n}'),
    ).toBeNull();
    expect(parseSiteLogImport('a.example.com {\n  import waf_protect\n}')).toBeNull();
    expect(parseSiteLogImport('')).toBeNull();
  });

  it('渲染出的 import 块能被解析回原设置', () => {
    for (const settings of [
      make(),
      make({ access: false }),
      make({ sizeMb: 1 }),
      make({ sizeMb: 1024 }),
    ]) {
      const block = [
        'a.example.com {',
        ...renderSiteImportBlock(settings, 'a').map((line) => `  ${line}`),
        '}',
      ].join('\n');
      expect(parseSiteLogImport(block)).toEqual({ name: 'a', settings });
    }
  });
});

describe('validateSiteLogSettings', () => {
  it('合法设置无错误', () => {
    expect(validateSiteLogSettings(make())).toEqual([]);
    expect(validateSiteLogSettings(make({ access: false }))).toEqual([]);
    expect(validateSiteLogSettings(make({ access: false, error: false }))).toEqual(
      [],
    );
    expect(validateSiteLogSettings(make({ sizeMb: 1 }))).toEqual([]);
    expect(validateSiteLogSettings(make({ sizeMb: 1024 }))).toEqual([]);
  });

  it('大小上限必须是 1..1024 的整数', () => {
    for (const sizeMb of [0, -1, 1025, 10.5]) {
      const errors = validateSiteLogSettings(make({ sizeMb }));
      expect(errors).toHaveLength(1);
      expect(errors[0]).toContain('1 至 1024');
    }
  });

  it('开访问 + 关错误报不可表达', () => {
    const errors = validateSiteLogSettings(make({ access: true, error: false }));
    expect(errors).toEqual([SITE_LOG_FORCED_ERROR_MESSAGE]);
    expect(errors[0]).toBe(
      'Caddy 无法单独关闭错误日志：开启访问日志会同时记录错误日志',
    );
  });

  it('两类问题同时存在时一并返回', () => {
    const errors = validateSiteLogSettings(make({ sizeMb: 0, error: false }));
    expect(errors).toHaveLength(2);
    expect(errors[1]).toBe(SITE_LOG_FORCED_ERROR_MESSAGE);
  });
});
