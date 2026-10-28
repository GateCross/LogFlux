<script lang="ts" setup>
import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Alert, Card } from 'antdv-next';

import SiteCreateModal from '../sites/SiteCreateModal.vue';
import ServiceList from './components/ServiceList.vue';
import { useServiceCatalogPage } from './composables/useServiceCatalogPage';

defineOptions({ name: 'CaddyCatalog' });

const createModalOpen = ref(false);

const {
  loadingConfig,
  configErrorMessage,
  siteCards,
  configRaw,
  configHash,
  keyword,
  siteMetricsMap,
  siteMetricsError,
  certList,
  filteredSites,
  fetchConfigSites,
  handleRefreshSites,
  openConfigWorkbench,
  openWaf,
  openAccessLogs,
  openSiteDetail,
  todayDisplay,
  setKeyword,
} = useServiceCatalogPage();
</script>

<template>
  <Page title="站点" description="查看站点与今日请求/拦截统计">
    <Alert
      v-if="configErrorMessage"
      class="mb-4"
      type="error"
      show-icon
      :message="configErrorMessage"
    />

    <Card variant="borderless">
      <ServiceList
        :loading="loadingConfig"
        :keyword="keyword"
        :site-cards-count="siteCards.length"
        :filtered-sites="filteredSites"
        :site-metrics-error="siteMetricsError"
        :site-metrics-map="siteMetricsMap"
        :cert-list="certList"
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
