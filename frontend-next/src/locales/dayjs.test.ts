/**
 * I18n 纯逻辑单元测试（任务 5.1）：语言归一化 / dayjs 同步 / 初始语言读取 / 语言持久化。
 *
 * 以「具体示例 / 边界情形」覆盖 `src/locales/dayjs.ts` 的核心行为；对「任意输入归一化恒合法」
 * 的普适性属性由属性测试（任务 5.2，Property 7）承担，避免重复。
 *
 * 说明：本测试仅覆盖框架无关的纯逻辑（不导入 `@umijs/max`），无刷新切换 `setLocale(lang,false)`
 * 与语言变更订阅属于运行时 wiring（`src/locales/index.ts` / `src/app.tsx`），由后续组件/集成
 * 测试覆盖。
 */
import dayjs from 'dayjs';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { serializePreferences, createDefaultPreferences, type UserPreferences } from '@/utils/preferences';
import {
  DEFAULT_LANG,
  PREFERENCES_STORAGE_KEY,
  getInitialLang,
  normalizeLang,
  persistLang,
  setDayjsLocale,
} from './dayjs';

describe('normalizeLang（语言归一化，Req 5.6）', () => {
  it('合法语言码原样返回', () => {
    expect(normalizeLang('zh-CN')).toBe('zh-CN');
    expect(normalizeLang('en-US')).toBe('en-US');
  });

  it('缺省 / 空串 / 非法值回退 zh-CN', () => {
    expect(normalizeLang(undefined)).toBe('zh-CN');
    expect(normalizeLang(null)).toBe('zh-CN');
    expect(normalizeLang('')).toBe('zh-CN');
    expect(normalizeLang('fr-FR')).toBe('zh-CN');
    expect(normalizeLang('zh')).toBe('zh-CN');
    expect(normalizeLang('EN-US')).toBe('zh-CN'); // 大小写敏感
    expect(normalizeLang(123)).toBe('zh-CN');
    expect(normalizeLang({})).toBe('zh-CN');
  });

  it('DEFAULT_LANG 为 zh-CN', () => {
    expect(DEFAULT_LANG).toBe('zh-CN');
  });
});

describe('setDayjsLocale（dayjs 同步，Req 5.4）', () => {
  afterEach(() => {
    dayjs.locale('en'); // 复位，避免跨用例污染
  });

  it('en-US → dayjs locale 切到 en', () => {
    expect(setDayjsLocale('en-US')).toBe('en-US');
    expect(dayjs.locale()).toBe('en');
  });

  it('zh-CN → dayjs locale 切到 zh-cn', () => {
    expect(setDayjsLocale('zh-CN')).toBe('zh-CN');
    expect(dayjs.locale()).toBe('zh-cn');
  });

  it('非法输入归一化为 zh-CN 并切到 zh-cn', () => {
    expect(setDayjsLocale('xx-YY')).toBe('zh-CN');
    expect(dayjs.locale()).toBe('zh-cn');
  });
});

describe('getInitialLang / persistLang（Preferences_Store 读写，Req 5.3 / 5.5）', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it('无持久化偏好时初始语言回退 zh-CN', () => {
    expect(getInitialLang()).toBe('zh-CN');
  });

  it('从持久化偏好读取初始语言', () => {
    const prefs: UserPreferences = { ...createDefaultPreferences(), lang: 'en-US' };
    window.localStorage.setItem(PREFERENCES_STORAGE_KEY, serializePreferences(prefs));
    expect(getInitialLang()).toBe('en-US');
  });

  it('持久化偏好非法时初始语言回退 zh-CN', () => {
    window.localStorage.setItem(PREFERENCES_STORAGE_KEY, '{ not valid json');
    expect(getInitialLang()).toBe('zh-CN');
  });

  it('persistLang 仅更新 lang 字段而保留其它偏好', () => {
    const prefs: UserPreferences = {
      ...createDefaultPreferences(),
      theme: { themeScheme: 'dark', themeColor: '#123456', grayscale: true, colourWeakness: false },
      lang: 'zh-CN',
    };
    window.localStorage.setItem(PREFERENCES_STORAGE_KEY, serializePreferences(prefs));

    expect(persistLang('en-US')).toBe('en-US');

    const raw = window.localStorage.getItem(PREFERENCES_STORAGE_KEY);
    expect(raw).not.toBeNull();
    const stored = JSON.parse(raw as string) as UserPreferences;
    expect(stored.lang).toBe('en-US');
    // 其它字段保持不变
    expect(stored.theme.themeColor).toBe('#123456');
    expect(stored.theme.themeScheme).toBe('dark');
    expect(stored.theme.grayscale).toBe(true);
  });

  it('persistLang 对非法输入归一化为 zh-CN 后写入', () => {
    expect(persistLang('bogus')).toBe('zh-CN');
    const stored = JSON.parse(window.localStorage.getItem(PREFERENCES_STORAGE_KEY) as string) as UserPreferences;
    expect(stored.lang).toBe('zh-CN');
  });

  it('getInitialLang 能读回 persistLang 写入的语言（往返）', () => {
    persistLang('en-US');
    expect(getInitialLang()).toBe('en-US');
  });
});
