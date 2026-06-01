/**
 * Preferences_Store 纯逻辑单元测试（任务 3.1）。
 *
 * 这里以「具体示例 / 边界情形」覆盖 serialize / deserialize / loadPreferences 与默认偏好的
 * 核心行为；对「所有合法偏好往返一致」与「所有非法输入回退默认」的普适性属性，
 * 由属性测试（任务 3.2 / 3.3，Property 15 / 16）承担，避免重复。
 */
import { describe, expect, it } from 'vitest';
import {
  createDefaultPreferences,
  DEFAULT_PREFERENCES,
  deserializePreferences,
  isValidPreferences,
  isValidThemeColor,
  loadPreferences,
  MAX_WATERMARK_TEXT_LENGTH,
  serializePreferences,
  type UserPreferences,
} from './preferences';

describe('serialize / deserialize 往返（示例与边界）', () => {
  it('对默认偏好往返后深度相等', () => {
    const original = createDefaultPreferences();
    expect(deserializePreferences(serializePreferences(original))).toEqual(original);
  });

  it('对含特殊 / Unicode 字符与最长字符串的水印文案往返后深度相等', () => {
    const original: UserPreferences = {
      theme: { themeScheme: 'dark', themeColor: 'rgba(0, 128, 255, 0.5)', grayscale: true, colourWeakness: false },
      lang: 'en-US',
      layout: { mode: 'horizontal', siderCollapse: true },
      watermark: { visible: true, text: '🌊 LogFlux 日志\t"流量"<&>'.padEnd(MAX_WATERMARK_TEXT_LENGTH, '★').slice(0, MAX_WATERMARK_TEXT_LENGTH) },
    };
    expect(deserializePreferences(serializePreferences(original))).toEqual(original);
  });

  it('对省略可选 watermark.text（空可选字段）的偏好往返后深度相等', () => {
    const original: UserPreferences = {
      theme: { themeScheme: 'auto', themeColor: '#fff', grayscale: false, colourWeakness: true },
      lang: 'zh-CN',
      layout: { mode: 'vertical-mix', siderCollapse: false },
      watermark: { visible: false },
    };
    const roundTripped = deserializePreferences(serializePreferences(original));
    expect(roundTripped).toEqual(original);
    expect('text' in roundTripped.watermark).toBe(false);
  });
});

describe('DEFAULT_PREFERENCES 与 createDefaultPreferences', () => {
  it('默认偏好自身合法', () => {
    expect(isValidPreferences(DEFAULT_PREFERENCES)).toBe(true);
  });

  it('createDefaultPreferences 每次返回独立副本（修改互不影响）', () => {
    const a = createDefaultPreferences();
    const b = createDefaultPreferences();
    a.theme.themeColor = '#000000';
    expect(b.theme.themeColor).not.toBe('#000000');
  });
});

describe('isValidThemeColor', () => {
  it('接受合法 hex / rgb / rgba', () => {
    for (const color of ['#fff', '#ffffff', '#abcd', '#11223344', 'rgb(0,0,0)', 'rgba(255, 255, 255, 0.3)']) {
      expect(isValidThemeColor(color)).toBe(true);
    }
  });

  it('拒绝空串 / 命名颜色 / 越界 / 非字符串', () => {
    for (const color of ['', 'red', '#12', 'rgb(256,0,0)', 'rgba(0,0,0,2)', '#xyzxyz', 123 as unknown]) {
      expect(isValidThemeColor(color)).toBe(false);
    }
  });
});

describe('loadPreferences 回退判定（Req 15.4 / 15.5）', () => {
  it('合法字符串 → 规范化偏好，usedDefault=false', () => {
    const original = createDefaultPreferences();
    const result = loadPreferences(serializePreferences(original));
    expect(result.usedDefault).toBe(false);
    expect(result.preferences).toEqual(original);
  });

  it('null / undefined / 空串 → 默认偏好，usedDefault=true', () => {
    for (const raw of [null, undefined, '']) {
      const result = loadPreferences(raw);
      expect(result.usedDefault).toBe(true);
      expect(result.preferences).toEqual(createDefaultPreferences());
    }
  });

  it('语法非法字符串（JSON.parse 抛错）→ 回退默认，不抛异常', () => {
    const result = loadPreferences('{not valid json');
    expect(result.usedDefault).toBe(true);
    expect(result.preferences).toEqual(createDefaultPreferences());
  });

  it('可解析但缺必需字段 → 回退默认', () => {
    const result = loadPreferences(JSON.stringify({ lang: 'zh-CN' }));
    expect(result.usedDefault).toBe(true);
  });

  it('可解析但取值越界（非法主色 / 未知枚举 / 超长水印）→ 回退默认', () => {
    const base = createDefaultPreferences();
    const badColor = { ...base, theme: { ...base.theme, themeColor: 'not-a-color' } };
    const badLang = { ...base, lang: 'fr-FR' };
    const badMode = { ...base, layout: { ...base.layout, mode: 'unknown-mode' } };
    const tooLongText = { ...base, watermark: { visible: true, text: 'x'.repeat(MAX_WATERMARK_TEXT_LENGTH + 1) } };
    for (const bad of [badColor, badLang, badMode, tooLongText]) {
      expect(loadPreferences(JSON.stringify(bad)).usedDefault).toBe(true);
    }
  });

  it('剔除合法对象中的多余键（返回规范化副本）', () => {
    const withExtra = { ...createDefaultPreferences(), extra: 'ignored', theme: { ...createDefaultPreferences().theme, junk: 1 } };
    const result = loadPreferences(JSON.stringify(withExtra));
    expect(result.usedDefault).toBe(false);
    expect(result.preferences).toEqual(createDefaultPreferences());
    expect('extra' in result.preferences).toBe(false);
    expect('junk' in result.preferences.theme).toBe(false);
  });
});
