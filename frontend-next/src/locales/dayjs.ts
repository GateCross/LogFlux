/**
 * dayjs 本地化同步 + 语言归一化 + 经 Preferences_Store 的语言持久化（任务 5.1）。
 *
 * 设计依据：design.md「Components and Interfaces ▸ I18n_Module」与「Correctness Properties ▸
 * Property 7（语言归一化与回退）」。
 *
 * 本模块**不依赖** UmiJS / React 运行时（仅依赖 dayjs、Preferences_Store 纯逻辑与
 * `window.localStorage`），从而可被属性测试（任务 5.2，Property 7）直接导入与覆盖。
 * 真正调用 Umi `setLocale(lang, false)` 的无刷新切换入口与语言变更事件订阅放在
 * `src/locales/index.ts`，由 `src/app.tsx` 在启动时装配。
 *
 * 支撑的需求 / 属性：
 *  - Req 5.3 初始语言读自 Preferences_Store（{@link getInitialLang}）。
 *  - Req 5.4 语言切换时同步 dayjs 本地化语言（{@link setDayjsLocale}）。
 *  - Req 5.5 切换时经 Preferences_Store 持久化所选语言（{@link persistLang}）。
 *  - Req 5.6 / Property 7：归一化输出恒属于 `{zh-CN, en-US}`，缺省 / 非法回退 `zh-CN`
 *    （{@link normalizeLang}）。
 */
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import 'dayjs/locale/en';
import { STORAGE_PREFIX } from '@/constants';
import { LANGS, loadPreferences, serializePreferences, type LangType } from '@/utils/preferences';

/**
 * 应用语言 → dayjs 内置 locale id 的映射。
 *
 * 迁移自旧 Vue 版 `frontend/src/locales/dayjs.ts`：`zh-CN → zh-cn`、`en-US → en`。
 */
const DAYJS_LOCALE_MAP: Record<LangType, string> = {
  'zh-CN': 'zh-cn',
  'en-US': 'en',
};

/** 默认语言（缺省 / 非法时的回退目标，Req 5.6）。 */
export const DEFAULT_LANG: LangType = 'zh-CN';

/**
 * Preferences_Store 在本地存储中的 key（带 `LF_` 前缀，Req 15.1）。
 *
 * 语言作为 `UserPreferences.lang` 字段持久化于此（与远端 `/api/user/preferences` 的
 * 偏好字符串同构）。完整的本地存储封装与远端同步事务见任务 3.4。
 */
export const PREFERENCES_STORAGE_KEY = `${STORAGE_PREFIX}preferences`;

/**
 * 语言归一化（Req 5.6 / Property 7）。
 *
 * 对任意输入（含 `undefined` / 空串 / 非法字符串 / 合法语言码），输出恒属于
 * `{ 'zh-CN', 'en-US' }`；当输入不是有效语言码时回退为 `zh-CN`。
 *
 * @param input 任意待归一化的语言输入
 * @returns 归一化后的合法语言码
 */
export function normalizeLang(input: unknown): LangType {
  if (typeof input === 'string' && (LANGS as readonly string[]).includes(input)) {
    return input as LangType;
  }
  return DEFAULT_LANG;
}

/**
 * 将 dayjs 的全局本地化语言同步为指定应用语言（Req 5.4）。
 *
 * 入参先经 {@link normalizeLang} 归一化，因此对非法 / 缺省输入会安全地切到 `zh-CN`。
 *
 * @param lang 目标应用语言（任意输入，内部归一化）
 * @returns 实际生效的归一化语言
 */
export function setDayjsLocale(lang?: unknown): LangType {
  const normalized = normalizeLang(lang);
  dayjs.locale(DAYJS_LOCALE_MAP[normalized]);
  return normalized;
}

/** 安全读取持久化偏好字符串；本地存储不可用时返回 `null`。 */
function readPreferencesRaw(): string | null {
  try {
    if (typeof window === 'undefined' || !window.localStorage) {
      return null;
    }
    return window.localStorage.getItem(PREFERENCES_STORAGE_KEY);
  } catch {
    return null;
  }
}

/** 安全写入持久化偏好字符串；本地存储不可用时静默忽略。 */
function writePreferencesRaw(raw: string): void {
  try {
    if (typeof window === 'undefined' || !window.localStorage) {
      return;
    }
    window.localStorage.setItem(PREFERENCES_STORAGE_KEY, raw);
  } catch {
    /* 本地存储不可用（隐私模式 / 配额超限）时忽略，不中断应用 */
  }
}

/**
 * 读取初始语言（Req 5.3 / 5.6）。
 *
 * 从 Preferences_Store 读取持久化偏好（经 `loadPreferences` 容错回退），取其 `lang` 字段
 * 并归一化。当无持久化偏好、偏好非法或语言字段非法时回退 `zh-CN`。
 *
 * @returns 归一化后的初始语言
 */
export function getInitialLang(): LangType {
  const { preferences } = loadPreferences(readPreferencesRaw());
  return normalizeLang(preferences.lang);
}

/**
 * 将所选语言经 Preferences_Store 持久化（Req 5.5）。
 *
 * 读取当前持久化偏好（容错回退默认），仅更新其 `lang` 字段后整体序列化写回本地存储，
 * 从而不破坏其它偏好字段。入参先归一化以保证写入的语言始终合法。
 *
 * @param lang 待持久化的语言（任意输入，内部归一化）
 * @returns 实际持久化的归一化语言
 */
export function persistLang(lang: unknown): LangType {
  const normalized = normalizeLang(lang);
  const { preferences } = loadPreferences(readPreferencesRaw());
  preferences.lang = normalized;
  writePreferencesRaw(serializePreferences(preferences));
  return normalized;
}
