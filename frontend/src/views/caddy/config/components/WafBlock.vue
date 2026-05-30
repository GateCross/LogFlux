<script setup lang="ts">
import SimpleWafPanel from './SimpleWafPanel.vue';
import WafIntegrationCard from './WafIntegrationCard.vue';
import type { useSimpleWafBlock } from '../composables/useSimpleWafBlock';

defineProps<{
  waf: ReturnType<typeof useSimpleWafBlock>;
}>();
</script>

<template>
  <div class="h-full min-h-0 overflow-auto">
    <NSpace vertical size="large">
      <SimpleWafPanel :waf="waf" />

      <WafIntegrationCard
        :loading="waf.wafIntegrationLoading.value"
        :submitting="waf.wafIntegrationSubmitting.value"
        :previewing="waf.wafIntegrationPreviewing.value"
        :unavailable="waf.wafIntegrationUnavailable.value"
        :status="waf.wafIntegrationStatus.value"
        :selected-sites="waf.selectedWafIntegrationSites.value"
        :preview-actions="waf.wafIntegrationPreviewActions.value"
        :on-refresh="waf.handleRefreshWafIntegrationState"
        :on-preview="waf.handlePreviewWafIntegration"
        :on-enable="waf.handleEnableWafIntegration"
        :on-disable="waf.handleDisableWafIntegration"
        :on-site-change="waf.handleWafIntegrationSiteChange"
      />
    </NSpace>
  </div>
</template>
