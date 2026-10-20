import {
  defineOverridesPreferences,
  definePreferencesExtension,
  preferences,
  updatePreferences,
} from '@vben/preferences';

/** 本地 logo 路径，随 public 目录一起发布 */
const LOCAL_LOGO = '/logo.svg';

interface WebAntdPreferencesExtension {
  defaultTableSize: number;
  enableFormFullscreen: boolean;
}

/**
 * LogFlux 项目配置覆盖
 */
export const overridesPreferences = defineOverridesPreferences({
  app: {
    name: import.meta.env.VITE_APP_TITLE || 'LogFlux',
    // 默认首页
    defaultHomePath: '/dashboard',
    // 启用 token 刷新
    enableRefreshToken: true,
  },
  logo: {
    // 不允许回落到外网图标
    source: LOCAL_LOGO,
  },
});

/**
 * 修正本地缓存里残留的外网资源地址。
 * 偏好设置以 localStorage 为优先，老版本用户缓存里存的是 CDN logo，
 * 不改写会继续请求外网。
 */
export function sanitizeCachedPreferences() {
  const { source, sourceDark } = preferences.logo;
  const isRemote = (value?: string) => !!value?.startsWith('http');
  if (!isRemote(source) && !isRemote(sourceDark)) {
    return;
  }
  updatePreferences({
    logo: {
      source: isRemote(source) ? LOCAL_LOGO : source,
      // defu 会忽略 undefined，清空必须用空串
      sourceDark: isRemote(sourceDark) ? '' : sourceDark,
    },
  });
}

export const preferencesExtension =
  definePreferencesExtension<WebAntdPreferencesExtension>({
    tabLabel: 'preferences.antd.tabLabel',
    title: 'preferences.antd.title',
    fields: [
      {
        component: 'switch',
        defaultValue: true,
        key: 'enableFormFullscreen',
        label: 'preferences.antd.fields.enableFormFullscreen.label',
        tip: 'preferences.antd.fields.enableFormFullscreen.tip',
      },
      {
        component: 'number',
        componentProps: {
          max: 200,
          min: 10,
          step: 10,
        },
        defaultValue: 20,
        key: 'defaultTableSize',
        label: 'preferences.antd.fields.defaultTableSize.label',
      },
    ],
  });
