import { computed, ref, watch } from 'vue';
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router';
import {
  SECURITY_OBSERVE_QUERY_KEYS,
  getSecurityDefaultTab,
  getSecurityMenuByTab,
  getSecurityMenuRouteName,
  getSecurityTabTitle,
  isSecurityMenuTabNavVisible,
  isSecurityTabVisible,
  pickRouteQueryValue,
  resolveSecurityMenuFromRoute,
  resolveSecurityTabFromRoute,
  type SecurityMenuKey,
  type SecurityTabKey
} from '../navigation';

interface UseSecurityNavigationOptions {
  route: RouteLocationNormalizedLoaded;
  router: Router;
}

export function useSecurityNavigation(options: UseSecurityNavigationOptions) {
  const { route, router } = options;

  const activeMenu = ref<SecurityMenuKey>('source');
  const activeTab = ref<SecurityTabKey>('source');

  const pageTitle = computed(() => `安全管理 · ${getSecurityTabTitle(activeTab.value)}`);
  const isMenuTabNavVisible = computed(() => isSecurityMenuTabNavVisible(activeMenu.value));

  function isTabVisible(tab: SecurityTabKey) {
    return isSecurityTabVisible(activeMenu.value, tab);
  }

  function syncNavigationStateFromRoute() {
    const nextMenu = resolveSecurityMenuFromRoute(String(route.name || ''), route.query.activeTab);
    const nextTab = resolveSecurityTabFromRoute(nextMenu, String(route.name || ''), route.query.activeTab);

    if (activeMenu.value !== nextMenu) {
      activeMenu.value = nextMenu;
    }

    if (activeTab.value !== nextTab) {
      activeTab.value = nextTab;
    }
  }

  async function navigateToSecurityTab(tab: SecurityTabKey, replace = false) {
    const menu = getSecurityMenuByTab(tab);
    const targetRouteName = getSecurityMenuRouteName(menu);
    if (!targetRouteName) {
      return;
    }

    const query: Record<string, string> = {};
    if (tab === 'observe') {
      SECURITY_OBSERVE_QUERY_KEYS.forEach(key => {
        const value = pickRouteQueryValue(route.query[key]);
        if (value) {
          query[key] = value;
        }
      });
    } else {
      const defaultTab = getSecurityDefaultTab(menu);
      if (tab !== defaultTab) {
        query.activeTab = tab;
      }
    }

    const navigationMethod = replace ? router.replace : router.push;
    await navigationMethod({
      name: targetRouteName as any,
      query
    });
  }

  watch(
    () => [route.name, route.query.activeTab],
    () => {
      syncNavigationStateFromRoute();
    },
    { immediate: true }
  );

  return {
    activeMenu,
    activeTab,
    pageTitle,
    isMenuTabNavVisible,
    isTabVisible,
    navigateToSecurityTab,
    syncNavigationStateFromRoute
  };
}
