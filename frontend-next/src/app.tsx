/**
 * Umi Max 运行时配置（app.tsx）。
 *
 * 当前仅装配 I18n_Module 相关运行时（任务 5.1）：
 *  - `locale.getLocale`：将**初始语言**读自 Preferences_Store（Req 5.3），缺省 / 非法回退
 *    `zh-CN`（Req 5.6）。Umi 在首次确定语言时会调用此函数。
 *  - `setupI18n()`：完成首屏 dayjs 同步，并订阅语言变更以在切换时同步 dayjs（Req 5.4）
 *    并经 Preferences_Store 持久化所选语言（Req 5.5）。
 *
 * 后续任务将在本文件扩展 `getInitialState`、`patchClientRoutes`（动态路由注入）、`layout`
 * 与路由守卫（任务 7.4 / 8.1）以及 `request` 拦截器装配。
 */
import type { RuntimeConfig } from '@umijs/max';
import { getInitialLang, setupI18n } from '@/locales';

// 应用启动时装配 I18n 运行时（dayjs 首屏同步 + 语言变更订阅）。
setupI18n();

/**
 * 国际化运行时扩展（Req 5.3 / 5.6）。
 *
 * 覆盖 Umi 默认的 `getLocale`（默认读 `localStorage.umi_locale` / 浏览器语言），改为以
 * Preferences_Store 持久化的语言作为初始语言来源。
 */
export const locale: RuntimeConfig['locale'] = {
  getLocale() {
    return getInitialLang();
  },
};
