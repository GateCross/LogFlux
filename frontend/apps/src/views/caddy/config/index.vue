<script lang="ts" setup>
import { Page } from '@vben/common-ui';
import { Alert, Card, Spin } from 'antdv-next';

import DockerDiscoveryModal from './DockerDiscoveryModal.vue';
import ServerToolbar from './components/ServerToolbar.vue';
import ModeStrip from './components/ModeStrip.vue';
import BlocksWorkbench from './components/BlocksWorkbench.vue';
// WafPanel 由配置管理 waf 视角与全局设置共用，放在 shared 下
import WafPanel from '../shared/components/WafPanel.vue';
import RawEditorPanel from './components/RawEditorPanel.vue';
import HistoryDrawer from './components/HistoryDrawer.vue';
import SavePreviewModal from './components/SavePreviewModal.vue';
import { useCaddyConfigPage } from './composables/useCaddyConfigPage';

defineOptions({ name: 'CaddyConfig' });

const {
  configErrorMessage,
  loadingConfig,
  mode,
  modeOptions,
  quickValidationErrors,
  configContent,
  previewConfig,
  dockerDiscoveryOpen,
  dockerDiscoveryScanning,
  dockerDiscoveryResult,
  serverToolbarProps,
  blocksWorkbenchProps,
  wafPanelProps,
  savePreviewProps,
  historyDrawerProps,
  openHistory,
  applyPreset,
  openDockerDiscovery,
  parseRawToBlocks,
  previewBeforeSave,
  handleModeChange,
  setActiveQuickSiteId,
  setWafPreviewVisible,
  setSavePreviewOpen,
  setHistoryDrawerVisible,
  setHistoryDetailVisible,
  setHistoryCompareVisible,
  setHistoryDiffOnly,
  addUpstream,
  removeUpstream,
  handleRefreshSiteMetrics,
  addQuickSite,
  openAccessLogs,
  duplicateQuickSite,
  removeQuickSite,
  saveWafConfig,
  previewWafConfig,
  applyWafConfig,
  fetchWafConfig,
  confirmSave,
  openHistoryDetail,
  openHistoryCompare,
  rollbackHistory,
  handleDockerDiscoveryScan,
  handleDiscoveryCommitDrafts,
  handleDiscoveryPreview,
  handleDiscoveryApply,
} = useCaddyConfigPage();
</script>

<template>
  <Page title="Caddy 配置">
    <div class="caddy-config-page">
      <Card variant="borderless" class="caddy-shell">
        <ServerToolbar
          v-bind="serverToolbarProps"
          @open-history="openHistory"
          @apply-preset="applyPreset"
          @parse-raw="parseRawToBlocks(true)"
          @save-raw="previewBeforeSave('raw')"
          @save-blocks="previewBeforeSave('blocks')"
        />

        <Alert
          v-if="configErrorMessage"
          class="mb-3"
          type="error"
          show-icon
          :message="configErrorMessage"
        />

        <Spin :spinning="loadingConfig">
          <ModeStrip :mode="mode" :mode-options="modeOptions" @change="handleModeChange" />

          <Alert
            v-if="mode === 'blocks' && quickValidationErrors.length"
            class="mb-3"
            :message="quickValidationErrors[0]"
            show-icon
            type="error"
          />

          <BlocksWorkbench
            v-if="mode === 'blocks'"
            v-bind="blocksWorkbenchProps"
            @update:active-quick-site-id="setActiveQuickSiteId"
            @add-upstream="addUpstream"
            @remove-upstream="removeUpstream"
            @refresh-metrics="handleRefreshSiteMetrics"
            @open-docker="openDockerDiscovery"
            @add-site="addQuickSite"
            @open-access-logs="(domains, e) => openAccessLogs(domains, e)"
            @duplicate-site="duplicateQuickSite"
            @remove-site="removeQuickSite"
          />

          <WafPanel
            v-else-if="mode === 'waf'"
            v-bind="wafPanelProps"
            @update:waf-preview-visible="setWafPreviewVisible"
            @save="saveWafConfig"
            @preview="previewWafConfig"
            @apply="applyWafConfig"
            @refresh="fetchWafConfig"
          />

          <RawEditorPanel
            v-else-if="mode === 'raw'"
            v-model="configContent"
          />

          <RawEditorPanel
            v-else
            :model-value="previewConfig"
            readonly
          />
        </Spin>
      </Card>
    </div>

    <DockerDiscoveryModal
      v-model:open="dockerDiscoveryOpen"
      :scanning="dockerDiscoveryScanning"
      :scan-result="dockerDiscoveryResult"
      @scan="handleDockerDiscoveryScan"
      @commit-drafts="handleDiscoveryCommitDrafts"
      @preview="handleDiscoveryPreview"
      @apply="handleDiscoveryApply"
    />

    <SavePreviewModal
      v-bind="savePreviewProps"
      @update:open="setSavePreviewOpen"
      @confirm="confirmSave"
    />

    <HistoryDrawer
      v-bind="historyDrawerProps"
      @update:open="setHistoryDrawerVisible"
      @update:history-detail-visible="setHistoryDetailVisible"
      @update:history-compare-visible="setHistoryCompareVisible"
      @update:history-diff-only="setHistoryDiffOnly"
      @open-detail="openHistoryDetail"
      @open-compare="openHistoryCompare"
      @rollback="rollbackHistory"
    />
  </Page>
</template>

<style scoped>
.caddy-config-page {
  padding: 16px;
}

.caddy-shell {
  min-height: calc(100vh - 140px);
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
