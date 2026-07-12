<script lang="ts" setup>
import { Page } from '@vben/common-ui';
import { Alert, Button, Card, Empty, Spin } from 'antdv-next';

import SiteWizardModal from './SiteWizardModal.vue';
import DockerDiscoveryModal from './DockerDiscoveryModal.vue';
import ServerToolbar from './components/ServerToolbar.vue';
import ModeStrip from './components/ModeStrip.vue';
import BlocksWorkbench from './components/BlocksWorkbench.vue';
import WafPanel from './components/WafPanel.vue';
import RawEditorPanel from './components/RawEditorPanel.vue';
import HistoryDrawer from './components/HistoryDrawer.vue';
import SavePreviewModal from './components/SavePreviewModal.vue';
import { useCaddyConfigPage } from './composables/useCaddyConfigPage';

defineOptions({ name: 'CaddyConfig' });

const {
  servers,
  selectedServerId,
  serversErrorMessage,
  configErrorMessage,
  loadingConfig,
  mode,
  modeOptions,
  quickValidationErrors,
  configContent,
  previewConfig,
  siteWizardOpen,
  dockerDiscoveryOpen,
  dockerDiscoveryScanning,
  dockerDiscoveryResult,
  mergedQuickFormModel,
  previewing,
  upstreamPoolOptions,
  serverToolbarProps,
  blocksWorkbenchProps,
  wafPanelProps,
  savePreviewProps,
  historyDrawerProps,
  handleServerChange,
  openAddServerModal,
  openEditServerModal,
  deleteCurrentServer,
  saveServer,
  fetchServers,
  handleRefreshServerStatus,
  openHistory,
  applyPreset,
  openSiteWizard,
  openDockerDiscovery,
  parseRawToBlocks,
  previewBeforeSave,
  handleModeChange,
  setServerModalVisible,
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
  handleWizardCommitDraft,
  handleWizardPreview,
  handleWizardApply,
} = useCaddyConfigPage();
</script>

<template>
  <Page title="Caddy 配置">
    <div class="caddy-config-page">
      <Card variant="borderless" class="caddy-shell">
        <ServerToolbar
          v-bind="serverToolbarProps"
          @update:selected-server-id="handleServerChange"
          @update:server-modal-visible="setServerModalVisible"
          @refresh-list="fetchServers"
          @refresh-status="handleRefreshServerStatus"
          @open-add="openAddServerModal"
          @open-edit="openEditServerModal"
          @delete-server="deleteCurrentServer"
          @open-history="openHistory"
          @apply-preset="applyPreset"
          @open-wizard="openSiteWizard"
          @open-docker="openDockerDiscovery"
          @parse-raw="parseRawToBlocks(true)"
          @save-raw="previewBeforeSave('raw')"
          @save-blocks="previewBeforeSave('blocks')"
          @save-server="saveServer"
        />

        <Alert
          v-if="serversErrorMessage"
          class="mb-3"
          type="error"
          show-icon
          :message="serversErrorMessage"
        />

        <Alert
          v-if="configErrorMessage"
          class="mb-3"
          type="error"
          show-icon
          :message="configErrorMessage"
        />

        <Spin :spinning="loadingConfig">
          <div v-if="servers.length === 0" class="empty-state">
            <Empty description="未找到 Caddy 服务器，请先添加节点">
              <Button type="primary" @click="openAddServerModal">添加服务器</Button>
            </Empty>
          </div>
          <template v-else>
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
              @open-wizard="openSiteWizard"
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
          </template>
        </Spin>
      </Card>
    </div>

    <SiteWizardModal
      v-model:open="siteWizardOpen"
      :base-form-model="mergedQuickFormModel"
      :previewing="previewing"
      :applying="previewing"
      :has-server="Boolean(selectedServerId)"
      :upstream-pool-options="upstreamPoolOptions"
      @commit-draft="handleWizardCommitDraft"
      @preview="handleWizardPreview"
      @apply="handleWizardApply"
    />

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

.empty-state {
  min-height: 420px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
