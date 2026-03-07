<script setup lang="ts">
import { computed } from 'vue';
import { NCheckbox, NCheckboxGroup } from 'naive-ui';
import type { WafIntegrationStatusResp } from '@/service/api/caddy-integration';

const props = defineProps<{
  loading: boolean;
  submitting: boolean;
  previewing: boolean;
  unavailable: boolean;
  status: WafIntegrationStatusResp | null;
  selectedSites: string[];
  previewActions: string[];
  onRefresh: () => void;
  onPreview: () => void;
  onEnable: () => void;
  onDisable: () => void;
  onSiteChange: (value: Array<string | number>) => void;
}>();

const siteOptions = computed(() => props.status?.availableSites || []);

function formatSiteSummary(list?: string[]) {
  if (!Array.isArray(list) || list.length === 0) {
    return '-';
  }
  if (list.length <= 2) {
    return list.join(' / ');
  }
  return `${list.slice(0, 2).join(' / ')} 等 ${list.length} 个`;
}
</script>

<template>
  <n-card :bordered="false" class="rounded-12px shadow-sm">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <div class="text-base font-semibold">Coraza 接入开关</div>
        <div class="mt-1 text-xs text-gray-500">自动补齐 order、waf_protect 片段，并按站点挂载或取消 `import waf_protect`。</div>
      </div>
      <div class="flex gap-2">
        <n-button size="small" :loading="loading" @click="props.onRefresh">刷新状态</n-button>
        <n-button size="small" :loading="previewing" @click="props.onPreview">预览变更</n-button>
        <n-button size="small" type="primary" :loading="submitting" @click="props.onEnable">一键接入</n-button>
        <n-button size="small" tertiary type="warning" :loading="submitting" @click="props.onDisable">取消接入</n-button>
      </div>
    </div>

    <n-grid cols="4" x-gap="12" y-gap="10" class="mt-4">
      <n-gi>
        <div class="text-xs text-gray-500">接入状态</div>
        <div class="text-sm font-medium">
          <n-tag :type="status?.integrated ? 'success' : 'warning'" :bordered="false">
            {{ status?.integrated ? '已接入' : '未接入' }}
          </n-tag>
        </div>
      </n-gi>
      <n-gi>
        <div class="text-xs text-gray-500">已挂载站点</div>
        <div class="text-sm font-medium">{{ formatSiteSummary(status?.importedSites) }}</div>
      </n-gi>
      <n-gi>
        <div class="text-xs text-gray-500">可选站点</div>
        <div class="text-sm font-medium">{{ formatSiteSummary(status?.availableSites) }}</div>
      </n-gi>
      <n-gi>
        <div class="text-xs text-gray-500">组件完整性</div>
        <div class="flex flex-wrap gap-2 text-sm font-medium">
          <n-tag size="small" :type="status?.orderReady ? 'success' : 'default'" :bordered="false">order</n-tag>
          <n-tag size="small" :type="status?.snippetReady ? 'success' : 'default'" :bordered="false">snippet</n-tag>
          <n-tag size="small" :type="status?.directiveReady ? 'success' : 'default'" :bordered="false">directives</n-tag>
        </div>
      </n-gi>
    </n-grid>

    <n-space vertical size="small" class="mt-4">
      <div class="text-xs text-gray-500">选择接入站点</div>
      <n-checkbox-group :value="selectedSites" @update:value="props.onSiteChange">
        <n-space wrap>
          <n-checkbox v-for="item in siteOptions" :key="item" :value="item" :label="item" />
        </n-space>
      </n-checkbox-group>
    </n-space>

    <n-alert v-if="unavailable" type="warning" :show-icon="true" class="mt-4">
      当前接入开关接口暂不可用，请确认后端已升级。
    </n-alert>
    <n-alert v-else-if="status?.message" type="info" :show-icon="true" class="mt-4">
      {{ status?.message }}
    </n-alert>

    <div v-if="previewActions.length" class="mt-4 rounded-8px bg-#fafafc p-3">
      <div class="text-xs font-semibold text-gray-700">最近一次预览动作</div>
      <div class="mt-2 flex flex-wrap gap-2">
        <n-tag v-for="item in previewActions" :key="item" size="small" type="info" :bordered="false">
          {{ item }}
        </n-tag>
      </div>
    </div>
  </n-card>
</template>
