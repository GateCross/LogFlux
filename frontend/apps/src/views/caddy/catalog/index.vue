<script lang="ts" setup>
import { Page } from '@vben/common-ui';
import { Alert, Card } from 'antdv-next';

import EditPanel from './components/EditPanel.vue';
import ServiceList from './components/ServiceList.vue';
import StatusActions from './components/StatusActions.vue';
import { useServiceCatalogPage } from './composables/useServiceCatalogPage';

defineOptions({ name: 'CaddyCatalog' });

const {
  servers,
  selectedServerId,
  loadingServers,
  loadingConfig,
  serversErrorMessage,
  configErrorMessage,
  siteCards,
  keyword,
  kindFilter,
  siteMetricsMap,
  loadingSiteMetrics,
  siteMetricsError,
  serverOptions,
  loadingServerStatus,
  serverStatusError,
  serverStatusLoaded,
  lastProbedAt,
  selectedServerStatus,
  serverStatusRows,
  filteredSites,
  summary,
  fetchConfigSites,
  handleRefreshServerStatus,
  handleRefreshSiteMetrics,
  handleServerChange,
  selectServerFromStatus,
  openConfigWorkbench,
  openWaf,
  openAccessLogs,
  openSiteInWorkbench,
  metricDisplay,
  setKeyword,
  setKindFilter,
} = useServiceCatalogPage();
</script>

<template>
  <Page title="服务目录" description="查看已登记节点上的站点目录与近窗指标">
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
        section="toolbar"
        :selected-server-id="selectedServerId"
        :server-options="serverOptions"
        :loading-servers="loadingServers"
        :selected-server-status="selectedServerStatus"
        :loading-server-status="loadingServerStatus"
        :loading-config="loadingConfig"
        :loading-site-metrics="loadingSiteMetrics"
        @update:selected-server-id="handleServerChange"
        @refresh-status="handleRefreshServerStatus"
        @refresh-sites="fetchConfigSites"
        @refresh-metrics="handleRefreshSiteMetrics"
        @open-workbench="openConfigWorkbench"
        @open-waf="openWaf"
      />
      <EditPanel
        section="secondary"
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
        :kind-filter="kindFilter"
        :summary="summary"
        :server-status-loaded="serverStatusLoaded"
        :site-metrics-error="siteMetricsError"
        :site-cards-count="siteCards.length"
        :filtered-sites="filteredSites"
        :site-metrics-map="siteMetricsMap"
        :metric-display="metricDisplay"
        @update:keyword="setKeyword"
        @update:kind-filter="setKindFilter"
        @open-site="openSiteInWorkbench"
        @open-access-logs="openAccessLogs"
        @open-waf="openWaf"
        @open-workbench="() => openConfigWorkbench()"
      />
    </Card>
  </Page>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
