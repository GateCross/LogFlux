import { type Ref, onMounted, watch } from 'vue';
import type { SecurityMenuKey, SecurityTabKey } from '../navigation';

interface UseSecurityRefreshOptions {
  activeMenu: Ref<SecurityMenuKey>;
  activeTab: Ref<SecurityTabKey>;
  refreshByTab: Record<SecurityTabKey, () => void>;
  refreshByMenu: Record<SecurityMenuKey, () => void>;
}

export function useSecurityRefresh(options: UseSecurityRefreshOptions) {
  const { activeMenu, activeTab, refreshByTab, refreshByMenu } = options;

  function refreshSecurityTab(tab: SecurityTabKey) {
    refreshByTab[tab]?.();
  }

  function refreshCurrentTab() {
    refreshSecurityTab(activeTab.value);
  }

  function refreshCurrentDomain() {
    refreshByMenu[activeMenu.value]?.();
  }

  watch(activeTab, value => {
    refreshSecurityTab(value);
  });

  watch(activeMenu, value => {
    refreshByMenu[value]?.();
  });

  onMounted(() => {
    refreshCurrentDomain();
  });

  return {
    refreshSecurityTab,
    refreshCurrentTab,
    refreshCurrentDomain
  };
}
