/**
 * Theme setup hook — manages dark/light/auto mode, algorithm selection,
 * and preference synchronization.
 */
import { useState, useEffect, useMemo } from 'react';
import { theme as antdTheme } from 'antd';
import { useModel } from '@umijs/max';

export function useThemeSetup() {
  const themeModel = useModel('theme');
  const appModel = useModel('app');

  const themeScheme = themeModel?.themeScheme ?? 'light';
  const themeColor = themeModel?.themeColor ?? '#1677ff';
  const initTheme = themeModel?.initTheme;
  const appPreferences = appModel?.preferences;

  // Detect OS dark mode preference
  const [prefersDark, setPrefersDark] = useState(() =>
    typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches,
  );

  // Reactive matchMedia listener — updates when OS theme changes
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;
    const mql = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => setPrefersDark(e.matches);
    mql.addEventListener('change', handler);
    return () => mql.removeEventListener('change', handler);
  }, []);

  // Resolve actual dark mode: auto follows OS, dark/light are fixed
  const isDark = useMemo(
    () => (themeScheme === 'auto' ? prefersDark : themeScheme === 'dark'),
    [themeScheme, prefersDark],
  );

  // Select Ant Design algorithm
  const algorithm = useMemo(
    () => (isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm),
    [isDark],
  );

  // Apply theme from preferences whenever they change
  useEffect(() => {
    if (appPreferences && initTheme) {
      initTheme(appPreferences);
    }
  }, [appPreferences, initTheme]);

  return { isDark, themeColor, algorithm };
}
