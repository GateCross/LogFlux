/**
 * 主题模型（Theme_Module，任务 6.1）。
 *
 * 设计依据：design.md「Theme_Module」，迁移自旧 Vue 版 `frontend/src/store/modules/theme/index.ts`。
 *
 * 职责：
 *  - 明暗方案管理（`themeScheme`）：light / dark / auto 三态切换。
 *  - 主题主色管理（`themeColor`）：校验并设置合法颜色值。
 *  - 无障碍辅助（`grayscale`、`colourWeakness`）：灰度模式与色弱模式。
 *  - 主题应用（`applyTheme`）：将主题状态同步到 `<html>` 元素的 `data-theme` 属性与 CSS 变量。
 *  - 主题初始化（`initTheme`）：从 UserPreferences 初始化，回退默认值。
 *
 * 支撑的需求：6.1, 6.2, 6.3, 6.7
 */
import { useState, useCallback } from 'react';
import { isValidThemeColor, type ThemeScheme, type UserPreferences } from '@/utils/preferences';

// ──────────────────────────────────────────────────────────────────────────
// 明暗方案循环顺序
// ──────────────────────────────────────────────────────────────────────────

const THEME_SCHEME_CYCLE: readonly ThemeScheme[] = ['light', 'dark', 'auto'];

/** 默认主题主色，与 `DEFAULT_PREFERENCES.theme.themeColor` 保持一致。 */
const DEFAULT_THEME_COLOR = '#646cff';

// ──────────────────────────────────────────────────────────────────────────
// Theme Model Hook
// ──────────────────────────────────────────────────────────────────────────

export default function useThemeModel() {
  const [themeScheme, setThemeScheme] = useState<ThemeScheme>('light');
  const [themeColor, setThemeColorState] = useState<string>(DEFAULT_THEME_COLOR);
  const [grayscale, setGrayscale] = useState(false);
  const [colourWeakness, setColourWeakness] = useState(false);

  // ──────────────────────────────────────────────────────────────────────
  // 明暗方案切换
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 循环切换明暗方案：light -> dark -> auto -> light。
   * 切换后自动调用 `applyTheme` 将新方案应用到文档。
   */
  const toggleThemeScheme = useCallback(() => {
    setThemeScheme((prev) => {
      const currentIndex = THEME_SCHEME_CYCLE.indexOf(prev);
      const nextIndex = (currentIndex + 1) % THEME_SCHEME_CYCLE.length;
      const next = THEME_SCHEME_CYCLE[nextIndex];

      // 应用新方案到文档
      applyThemeToDocument(next, themeColor, grayscale, colourWeakness);

      return next;
    });
  }, [themeColor, grayscale, colourWeakness]);

  // ──────────────────────────────────────────────────────────────────────
  // 主题主色设置
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 校验并设置主题主色（Req 6.3）。
   *
   * 使用 `isValidThemeColor` 校验颜色值合法性；非法值被拒绝并输出警告，
   * 不会更新状态。合法值写入状态并立即应用到文档。
   *
   * @param color 目标颜色值（十六进制或 rgb/rgba 格式）
   * @returns 是否设置成功
   */
  const setThemeColor = useCallback(
    (color: string): boolean => {
      if (!isValidThemeColor(color)) {
        console.warn(`[ThemeModel] Invalid theme color: "${color}"`);
        return false;
      }

      setThemeColorState(color);
      applyThemeToDocument(themeScheme, color, grayscale, colourWeakness);
      return true;
    },
    [themeScheme, grayscale, colourWeakness],
  );

  // ──────────────────────────────────────────────────────────────────────
  // 主题应用到文档
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 将主题状态应用到 `<html>` 元素（Req 6.1、6.2）。
   *
   * - 设置 `data-theme` 属性：'dark' 或 'light'（auto 时根据系统偏好判定）。
   * - 设置 `--primary-color` CSS 自定义属性。
   * - 应用灰度滤镜（`grayscale`）与色弱滤镜（`colourWeakness`）。
   *
   * 可传入覆盖参数（由 `initTheme` 使用），否则使用当前状态值。
   */
  const applyTheme = useCallback(
    (overrides?: {
      scheme?: ThemeScheme;
      color?: string;
      grayscale?: boolean;
      colourWeakness?: boolean;
    }) => {
      const scheme = overrides?.scheme ?? themeScheme;
      const color = overrides?.color ?? themeColor;
      const gs = overrides?.grayscale ?? grayscale;
      const cw = overrides?.colourWeakness ?? colourWeakness;

      applyThemeToDocument(scheme, color, gs, cw);
    },
    [themeScheme, themeColor, grayscale, colourWeakness],
  );

  // ──────────────────────────────────────────────────────────────────────
  // 主题初始化
  // ──────────────────────────────────────────────────────────────────────

  /**
   * 从 UserPreferences 初始化主题状态（Req 6.7）。
   *
   * 读取偏好对象中的 `theme` 字段，更新全部主题状态后立即应用到文档。
   * 通常在应用启动时或用户偏好加载完成后调用。
   */
  const initTheme = useCallback((preferences: UserPreferences) => {
    const { theme } = preferences;

    setThemeScheme(theme.themeScheme);
    setThemeColorState(theme.themeColor);
    setGrayscale(theme.grayscale);
    setColourWeakness(theme.colourWeakness);

    applyThemeToDocument(
      theme.themeScheme,
      theme.themeColor,
      theme.grayscale,
      theme.colourWeakness,
    );
  }, []);

  return {
    // 状态
    themeScheme,
    themeColor,
    grayscale,
    colourWeakness,

    // 方法
    toggleThemeScheme,
    setThemeColor,
    applyTheme,
    initTheme,
  };
}

// ──────────────────────────────────────────────────────────────────────────
// 文档级主题应用（纯副作用，不依赖 React 状态）
// ──────────────────────────────────────────────────────────────────────────

/**
 * 将主题参数直接应用到 `document.documentElement`。
 *
 * 提取为独立函数以便在 `useState` 回调、`useCallback` 闭包及 `initTheme` 中
 * 统一复用，避免多处重复的 DOM 操作逻辑。
 *
 * - `data-theme`：'dark' | 'light'（auto 时通过 `matchMedia` 检测系统偏好）。
 * - `--primary-color`：主题主色 CSS 变量。
 * - `filter`：灰度（`grayscale(1)`）与色弱（`invert(80%)`）无障碍滤镜。
 */
function applyThemeToDocument(
  scheme: ThemeScheme,
  color: string,
  grayscale: boolean,
  colourWeakness: boolean,
): void {
  if (typeof document === 'undefined') {
    return;
  }

  const html = document.documentElement;

  // 解析实际明暗方案（auto 时跟随系统偏好）
  let resolvedScheme: 'light' | 'dark';
  if (scheme === 'auto') {
    const prefersDark =
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-color-scheme: dark)').matches;
    resolvedScheme = prefersDark ? 'dark' : 'light';
  } else {
    resolvedScheme = scheme;
  }

  // 设置 data-theme 属性，供 CSS 选择器 `[data-theme="dark"]` 使用
  html.setAttribute('data-theme', resolvedScheme);

  // 设置主题主色 CSS 变量
  html.style.setProperty('--primary-color', color);

  // 构建无障碍滤镜
  const filters: string[] = [];
  if (grayscale) {
    filters.push('grayscale(1)');
  }
  if (colourWeakness) {
    filters.push('invert(80%)');
  }
  html.style.filter = filters.length > 0 ? filters.join(' ') : '';
}
