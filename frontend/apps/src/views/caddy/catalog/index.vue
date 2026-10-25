<script lang="ts" setup>
import { ref } from 'vue';
import { Page } from '@vben/common-ui';
import { Alert, Card } from 'antdv-next';

import SiteCreateModal from '../sites/SiteCreateModal.vue';
import EditPanel from './components/EditPanel.vue';
import ServiceList from './components/ServiceList.vue';
import StatusActions from './components/StatusActions.vue';
import { useServiceCatalogPage } from './composables/useServiceCatalogPage';

defineOptions({ name: 'CaddyCatalog' });

const createModalOpen = ref(false);

const {
  servers,
  selectedServerId,
  loadingServers,
  loadingConfig,
  serversErrorMessage,
  configErrorMessage,
  siteCards,
  configRaw,
  configHash,
  keyword,
  siteMetricsMap,
  siteMetricsError,
  certList,
  wafStatus,
  serverOptions,
  loadingServerStatus,
  serverStatusError,
  serverStatusLoaded,
  lastProbedAt,
  selectedServerStatus,
  serverStatusRows,
  filteredSites,
  fetchConfigSites,
  handleRefreshServerStatus,
  handleRefreshSites,
  handleServerChange,
  selectServerFromStatus,
  openConfigWorkbench,
  openWaf,
  openAccessLogs,
  openSiteDetail,
  todayDisplay,
  setKeyword,
} = useServiceCatalogPage();
</script>

<template>
  <Page title="站点" description="查看已登记节点上的站点与今日请求/拦截统计">
    <Alert
      v-if="serversErrorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="serversErrorMessage"
    />
    <Alert
      v-if="configErrorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="configErrorMessage"
    />

    <Card variant="borderless" class="mb-4">
      <EditPanel
        :selected-server-id="selectedServerId"
        :server-options="serverOptions"
        :loading-servers="loadingServers"
        :selected-server-status="selectedServerStatus"
        :loading-server-status="loadingServerStatus"
        :loading-config="loadingConfig"
        @update:selected-server-id="handleServerChange"
        @refresh-status="handleRefreshServerStatus"
        @open-workbench="openConfigWorkbench"
        @open-waf="openWaf"
      />
    </Card>

    <Card variant="borderless" class="mb-4">
      <StatusActions
        :servers-count="servers.length"
        :last-probed-at="lastProbedAt"
        :server-status-error="serverStatusError"
        :server-status-loaded="serverStatusLoaded"
        :selected-server-id="selectedServerId"
        :server-status-rows="serverStatusRows"
        @select-server="selectServerFromStatus"
        @open-workbench="() => openConfigWorkbench()"
      />
    </Card>

    <Card variant="borderless">
      <ServiceList
        :selected-server-id="selectedServerId"
        :loading="loadingConfig"
        :keyword="keyword"
        :site-cards-count="siteCards.length"
        :filtered-sites="filteredSites"
        :site-metrics-error="siteMetricsError"
        :site-metrics-map="siteMetricsMap"
        :cert-list="certList"
        :waf-status="wafStatus"
        :today-display="todayDisplay"
        @update:keyword="setKeyword"
        @refresh="handleRefreshSites"
        @add-site="createModalOpen = true"
        @open-site="openSiteDetail"
        @open-access-logs="openAccessLogs"
        @open-waf="openWaf"
        @open-workbench="() => openConfigWorkbench()"
      />
    </Card>

    <SiteCreateModal
      v-model:open="createModalOpen"
      :server-id="selectedServerId"
      :config="configRaw"
      :base-hash="configHash || undefined"
      @created="fetchConfigSites"
    />
  </Page>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
