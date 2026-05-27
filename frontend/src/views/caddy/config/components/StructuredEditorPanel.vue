<script setup lang="ts">
import type { CaddyFormModel, Site } from '../types';
import SiteListPanel from './SiteListPanel.vue';
import SiteEditorPanel from './SiteEditorPanel.vue';
import UpstreamManager from './UpstreamManager.vue';

type ValidationError = {
  id: string;
  message: string;
  siteId?: string;
  routeId?: string;
  tab?: 'basic' | 'routes' | 'advanced';
};

const props = defineProps<{
  formModel: CaddyFormModel;
  focusRouteId: string | null;
  sidebarWidth: number;
  structuredAvailable: boolean;
  validationErrors: ValidationError[];
  globalRawChanged: boolean;
  globalPreviewExpanded: boolean;
  globalPreviewText: string;
  onApplyPreset: () => void;
  onImportRawToStructured: () => void;
  onOpenPreviewModal: () => void;
  onToggleGlobalPreview: () => void;
  onOpenGlobalModal: () => void;
  onFocusValidationError: (item: ValidationError) => void;
  onStartResize: (event: MouseEvent) => void;
  onAddSite: () => void;
  onDuplicateSite: (id: string) => void;
  onRemoveSite: (id: string) => void;
  onMoveSite: (id: string, direction: 'up' | 'down') => void;
}>();

const activeSiteId = defineModel<string | null>('activeSiteId', { required: true });
const activeSite = defineModel<Site | null>('activeSite', { required: true });
const activeTab = defineModel<'basic' | 'routes' | 'advanced'>('activeTab', { required: true });
</script>

<template>
  <div
    class="caddy-split h-full min-h-0 min-w-0 flex flex-col overflow-hidden lg:flex-row"
    :style="{ '--sidebar-width': props.sidebarWidth + 'px' }"
  >
    <div class="caddy-sidebar min-w-0 flex-shrink-0">
      <SiteListPanel
        class="h-full"
        :sites="props.formModel.sites"
        :active-id="activeSiteId"
        @select="activeSiteId = $event"
        @add="props.onAddSite"
        @duplicate="props.onDuplicateSite"
        @remove="props.onRemoveSite"
        @move="props.onMoveSite"
      />
    </div>
    <div class="caddy-resizer hidden lg:block" @mousedown="props.onStartResize"></div>
    <div class="caddy-main-panel min-h-0 min-w-0 flex flex-col flex-1 gap-3 overflow-auto">
      <div class="flex flex-wrap items-center gap-2">
        <NButton size="small" @click="props.onApplyPreset">应用默认模板</NButton>
        <NButton size="small" @click="props.onImportRawToStructured">从原始配置解析</NButton>
        <NButton size="small" @click="props.onOpenPreviewModal">预览原始 Caddyfile</NButton>
      </div>
      <NAlert v-if="!props.structuredAvailable" type="warning" title="结构化配置未加载" class="mb-2">
        当前服务器未保存结构化配置，可通过“从原始配置解析”或“应用默认模板”生成。
      </NAlert>
      <NCard size="small" :bordered="false" class="bg-white">
        <template #header>全局配置（原样保留）</template>
        <template #header-extra>
          <div class="flex items-center gap-2">
            <NTag v-if="props.globalRawChanged" type="warning" size="small">未保存</NTag>
            <NButton size="tiny" @click="props.onToggleGlobalPreview">
              {{ props.globalPreviewExpanded ? '收起' : '展开' }}
            </NButton>
            <NButton size="tiny" @click="props.onOpenGlobalModal">查看/编辑</NButton>
          </div>
        </template>
        <pre
          class="global-preview cursor-pointer"
          :class="{ expanded: props.globalPreviewExpanded }"
          @click="props.onOpenGlobalModal"
          v-text="props.globalPreviewText || '未配置全局 options 块'"
        />
        <div class="mt-2 text-xs text-gray-500">该区域将原样拼接到生成的 Caddyfile 顶部。</div>
      </NCard>
      <NAlert v-if="props.validationErrors.length" type="error" title="配置校验错误" class="mb-2">
        <ul class="list-disc pl-4">
          <li v-for="item in props.validationErrors" :key="item.id">
            <a
              v-if="item.siteId"
              class="cursor-pointer text-blue-600 hover:underline"
              @click.prevent="props.onFocusValidationError(item)"
            >
              {{ item.message }}
            </a>
            <span v-else>{{ item.message }}</span>
          </li>
        </ul>
      </NAlert>
      <SiteEditorPanel v-model:site="activeSite" v-model:tab="activeTab" :focus-route-id="props.focusRouteId" />
      <NCollapse class="mt-2">
        <NCollapseItem title="上游池管理" name="upstreams">
          <UpstreamManager :upstreams="props.formModel.upstreams" />
        </NCollapseItem>
      </NCollapse>
    </div>
  </div>
</template>

<style scoped>
.caddy-split {
  gap: 12px;
}

.caddy-sidebar {
  width: 100%;
  min-height: 0;
}

.caddy-main-panel {
  padding-right: 4px;
}

.global-preview {
  max-height: 140px;
  overflow: hidden;
  white-space: pre-wrap;
  background: #f8fafc;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 12px;
  color: #475569;
}

.global-preview.expanded {
  max-height: 520px;
  overflow: auto;
}

@media (min-width: 1024px) {
  .caddy-sidebar {
    width: var(--sidebar-width);
    min-width: 240px;
  }

  .caddy-resizer {
    width: 6px;
    cursor: col-resize;
    border-radius: 999px;
    background: linear-gradient(180deg, #e2e8f0, #cbd5f5, #e2e8f0);
    align-self: stretch;
  }
}
</style>
