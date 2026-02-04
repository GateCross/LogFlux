<script setup lang="ts">
import SiteListPanel from './SiteListPanel.vue';
import SiteEditorPanel from './SiteEditorPanel.vue';
import UpstreamManager from './UpstreamManager.vue';
import type { CaddyFormModel, Site } from '../types';

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
  <div class="h-full flex flex-col lg:flex-row overflow-hidden min-w-0 caddy-split" :style="{ '--sidebar-width': props.sidebarWidth + 'px' }">
    <div class="caddy-sidebar flex-shrink-0 min-w-0">
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
    <div class="flex-1 min-w-0 flex flex-col gap-3 overflow-auto">
      <div class="flex flex-wrap gap-2 items-center">
        <n-button size="small" @click="props.onApplyPreset">应用默认模板</n-button>
        <n-button size="small" @click="props.onImportRawToStructured">从原始配置解析</n-button>
        <n-button size="small" @click="props.onOpenPreviewModal">预览原始 Caddyfile</n-button>
      </div>
      <n-alert v-if="!props.structuredAvailable" type="warning" title="结构化配置未加载" class="mb-2">
        当前服务器未保存结构化配置，可通过“从原始配置解析”或“应用默认模板”生成。
      </n-alert>
      <n-card size="small" :bordered="false" class="bg-white">
        <template #header>全局配置（原样保留）</template>
        <template #header-extra>
          <div class="flex items-center gap-2">
            <n-tag v-if="props.globalRawChanged" type="warning" size="small">未保存</n-tag>
            <n-button size="tiny" @click="props.onToggleGlobalPreview">
              {{ props.globalPreviewExpanded ? '收起' : '展开' }}
            </n-button>
            <n-button size="tiny" @click="props.onOpenGlobalModal">查看/编辑</n-button>
          </div>
        </template>
        <pre
          class="global-preview cursor-pointer"
          :class="{ expanded: props.globalPreviewExpanded }"
          @click="props.onOpenGlobalModal"
          v-text="props.globalPreviewText || '未配置全局 options 块'"
        />
        <div class="text-xs text-gray-500 mt-2">该区域将原样拼接到生成的 Caddyfile 顶部。</div>
      </n-card>
      <n-alert v-if="props.validationErrors.length" type="error" title="配置校验错误" class="mb-2">
        <ul class="list-disc pl-4">
          <li v-for="item in props.validationErrors" :key="item.id">
            <a
              v-if="item.siteId"
              class="text-blue-600 hover:underline cursor-pointer"
              @click.prevent="props.onFocusValidationError(item)"
            >
              {{ item.message }}
            </a>
            <span v-else>{{ item.message }}</span>
          </li>
        </ul>
      </n-alert>
      <SiteEditorPanel v-model:site="activeSite" v-model:tab="activeTab" :focus-route-id="props.focusRouteId" />
      <n-collapse class="mt-2">
        <n-collapse-item title="上游池管理" name="upstreams">
          <UpstreamManager :upstreams="props.formModel.upstreams" />
        </n-collapse-item>
      </n-collapse>
    </div>
  </div>
</template>

<style scoped>
.caddy-split {
  gap: 12px;
}

.caddy-sidebar {
  width: 100%;
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
