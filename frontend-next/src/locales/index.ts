/**
 * I18n_Module 运行时入口（任务 5.1）。
 *
 * 设计依据：design.md「Components and Interfaces ▸ I18n_Module」。
 *
 * 职责：
 *  - 暴露 {@link changeLang}：经 Umi `setLocale(lang, false)` **无刷新**切换当前视图语言（Req 5.2）。
 *  - 通过 {@link setupI18n} 订阅语言变更，切换时**同步 dayjs**（Req 5.4）并经 **Preferences_Store
 *    持久化**所选语言（Req 5.5）；该订阅同时覆盖经内置 `<SelectLang />` 等直接调用 `setLocale`
 *    的切换路径，使持久化与 dayjs 同步成为「单一真源」。
 *  - 初始语言读自 Preferences_Store 的逻辑见 `src/locales/dayjs.ts ▸ getInitialLang` 与
 *    `src/app.tsx` 的 `locale.getLocale` 运行时覆盖（Req 5.3 / 5.6）。
 *  - 缺键回退 `zh-CN`（Req 5.7）由 en-US 文案默认导出合并 zh-CN 基准实现，见 `src/locales/en-US.ts`。
 *
 * 说明：`src/locales/{zh-CN,en-US}.ts` 为「约定式」文案文件，由 Umi `locale` 插件自动加载，
 * 无需在此手动注册。
 */
import { setLocale } from '@umijs/max';
import type { LangType } from '@/utils/preferences';
import { LANGS } from '@/utils/preferences';
import { getInitialLang, normalizeLang, persistLang, setDayjsLocale } from './dayjs';

/**
 * Umi locale 插件用于持久化「当前生效语言」的 localStorage key（`useLocalStorage: true`）。
 *
 * `setLocale(lang, false)` 会在派发 `languagechange` 事件**之前**将新语言写入该 key，
 * 因此语言变更回调读取此 key 可获得**最新**语言（此时若改用我们覆盖的 `getLocale` 会读到
 * 尚未更新的 Preferences_Store 旧值）。该 key 名为 Umi 文档约定的稳定值。
 */
const UMI_LOCALE_STORAGE_KEY = 'umi_locale';

export { getInitialLang, normalizeLang } from './dayjs';
export type { LangType } from '@/utils/preferences';

/** 语言下拉选项（供语言切换 UI 使用，标签迁移自旧 Vue 版 `localeOptions`）。 */
export const LANG_OPTIONS: readonly { label: string; value: LangType }[] = [
  { label: '中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' },
];

/**
 * 无刷新切换界面语言（Req 5.2）。
 *
 * 入参经 {@link normalizeLang} 归一化后调用 Umi `setLocale(lang, false)`：当前视图内全部
 * 已注册文案键即时更新为所选语言，且不重新加载页面。dayjs 同步与语言持久化由
 * {@link setupI18n} 订阅的语言变更回调统一处理（Req 5.4 / 5.5）。
 *
 * @param lang 目标语言（任意输入，内部归一化）
 * @returns 实际生效的归一化语言
 */
export function changeLang(lang: unknown): LangType {
  const normalized = normalizeLang(lang);
  // 第二参数 false：无刷新切换（design.md I18n_Module，Req 5.2）。
  setLocale(normalized, false);
  return normalized;
}

/** 读取 Umi 当前生效语言并归一化（兜底回退 `zh-CN`）。 */
function getActiveLang(): LangType {
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      return normalizeLang(window.localStorage.getItem(UMI_LOCALE_STORAGE_KEY));
    }
  } catch {
    /* 本地存储不可用时回退默认 */
  }
  return normalizeLang(undefined);
}

/** 标记 setupI18n 是否已装配，避免重复订阅事件。 */
let i18nInitialized = false;

/**
 * 语言变更回调：同步 dayjs 本地化语言（Req 5.4）并经 Preferences_Store 持久化（Req 5.5）。
 *
 * Umi 在无刷新切换语言时会向 `window` 派发 `languagechange` 事件，并已在派发**之前**将新语言
 * 写入其内部的 `umi_locale` 存储；此处经 {@link getActiveLang} 读取该最新语言后统一处理 dayjs
 * 同步与持久化，因而无论切换来自 {@link changeLang} 还是内置 `<SelectLang />`，行为一致。
 */
function handleLanguageChange(): void {
  const active = getActiveLang();
  setDayjsLocale(active);
  persistLang(active);
}

/**
 * 装配 I18n 运行时（在应用启动时调用一次，见 `src/app.tsx`）。
 *
 * - 依据初始语言完成首屏 dayjs 同步（Req 5.4）。
 * - 订阅 `languagechange` 事件：后续每次语言切换都同步 dayjs 并持久化语言（Req 5.4 / 5.5）。
 *
 * 幂等：重复调用不会重复订阅。
 */
export function setupI18n(): void {
  // 首屏：将 dayjs 同步到初始语言（初始语言来自 Preferences_Store，见 getInitialLang）。
  setDayjsLocale(getInitialLang());

  if (i18nInitialized || typeof window === 'undefined' || typeof window.addEventListener !== 'function') {
    return;
  }
  i18nInitialized = true;
  window.addEventListener('languagechange', handleLanguageChange);
}

/** 支持的语言集合（Req 5.1，复用 Preferences_Store 的单一定义）。 */
export { LANGS };
