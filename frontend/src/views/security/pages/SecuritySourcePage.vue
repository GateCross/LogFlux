<script setup lang="ts">
import { computed } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { NCheckbox, NCheckboxGroup } from 'naive-ui';
import type { WafEngineStatusResp, WafSourceItem } from '@/service/api/caddy-source';
import type { WafIntegrationStatusResp } from '@/service/api/caddy-integration';
import SourceTabContent from '../tabs/SourceTabContent.vue';

const props = defineProps<{
  pageTitle: string;
  integrationLoading: boolean;
  integrationSubmitting: boolean;
  integrationPreviewing: boolean;
  integrationUnavailable: boolean;
  integrationStatus: WafIntegrationStatusResp | null;
  selectedIntegrationSites: string[];
  handleRefreshIntegrationStatus: () => void;
  handlePreviewIntegration: () => void | Promise<void>;
  handleEnableIntegration: () => void | Promise<void>;
  handleDisableIntegration: () => void | Promise<void>;
  handleIntegrationSiteChange: (value: Array<string | number>) => void;
  integrationPreviewActions: string[];
  engineLoading: boolean;
  engineChecking: boolean;
  engineUnavailable: boolean;
  engineStatus: WafEngineStatusResp | null;
  handleRefreshEngineStatus: () => void;
  handleCheckEngine: () => void | Promise<void>;
  displayEngineValue: (value: unknown) => string;
  sourceQuery: { name: string };
  sourceColumns: DataTableColumns<WafSourceItem>;
  sourceTable: WafSourceItem[];
  sourceLoading: boolean;
  sourcePagination: PaginationProps;
  tableFixedHeight: number;
  fetchSources: () => void | Promise<void>;
  resetSourceQuery: () => void;
  handleAddSource: () => void;
  openUploadModal: () => void;
  handleSourcePageChange: (page: number) => void;
  handleSourcePageSizeChange: (pageSize: number) => void;
}>();

const integrationSiteOptions = computed(() => props.integrationStatus?.availableSites || []);

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
  <div class="flex flex-col gap-3">
    <NAlert type="info" :show-icon="true" class="rounded-8px">
      <template #header>{{ pageTitle }}</template>
      <div>
        CRS 支持在线同步（含检查）、上传、激活与回滚；Coraza 引擎依赖 Caddy 二进制，仅提供 GitHub Release
        版本检查，不支持在线替换引擎。
      </div>
    </NAlert>

    <NCard :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">Coraza 接入开关</div>
          <div class="mt-1 text-xs text-gray-500">
            首次接入会自动补齐全局 order、waf_protect 片段，并按站点挂载 import waf_protect。
          </div>
        </div>
        <div class="flex gap-2">
          <NButton size="small" :loading="integrationLoading" @click="handleRefreshIntegrationStatus">
            刷新接入状态
          </NButton>
          <NButton size="small" :loading="integrationPreviewing" @click="handlePreviewIntegration">预览变更</NButton>
          <NButton size="small" type="primary" :loading="integrationSubmitting" @click="handleEnableIntegration">
            一键接入
          </NButton>
          <NButton
            size="small"
            tertiary
            type="warning"
            :loading="integrationSubmitting"
            @click="handleDisableIntegration"
          >
            取消接入
          </NButton>
        </div>
      </div>

      <NGrid cols="4" x-gap="12" y-gap="10" class="mt-4">
        <NGi>
          <div class="text-xs text-gray-500">接入状态</div>
          <div class="text-sm font-medium">
            <NTag :type="integrationStatus?.integrated ? 'success' : 'warning'" :bordered="false">
              {{ integrationStatus?.integrated ? '已接入' : '未接入' }}
            </NTag>
          </div>
        </NGi>
        <NGi>
          <div class="text-xs text-gray-500">已挂载站点</div>
          <div class="text-sm font-medium">
            {{ formatSiteSummary(integrationStatus?.importedSites) }}
          </div>
        </NGi>
        <NGi>
          <div class="text-xs text-gray-500">可选站点</div>
          <div class="text-sm font-medium">
            {{ formatSiteSummary(integrationStatus?.availableSites) }}
          </div>
        </NGi>
        <NGi>
          <div class="text-xs text-gray-500">组件完整性</div>
          <div class="flex flex-wrap gap-2 text-sm font-medium">
            <NTag size="small" :type="integrationStatus?.orderReady ? 'success' : 'default'" :bordered="false">
              order
            </NTag>
            <NTag size="small" :type="integrationStatus?.snippetReady ? 'success' : 'default'" :bordered="false">
              snippet
            </NTag>
            <NTag size="small" :type="integrationStatus?.directiveReady ? 'success' : 'default'" :bordered="false">
              directives
            </NTag>
          </div>
        </NGi>
      </NGrid>

      <NSpace vertical size="small" class="mt-4">
        <div class="text-xs text-gray-500">选择要接入的站点</div>
        <NCheckboxGroup :value="selectedIntegrationSites" @update:value="handleIntegrationSiteChange">
          <NSpace wrap>
            <NCheckbox v-for="item in integrationSiteOptions" :key="item" :value="item" :label="item" />
          </NSpace>
        </NCheckboxGroup>
      </NSpace>

      <NAlert v-if="integrationUnavailable" type="warning" :show-icon="true" class="mt-4">
        当前接入开关接口暂不可用，请确认后端已升级到最新版本。
      </NAlert>
      <NAlert v-else-if="integrationStatus?.message" type="info" :show-icon="true" class="mt-4">
        {{ integrationStatus?.message }}
      </NAlert>

      <div v-if="integrationPreviewActions.length" class="mt-4 rounded-8px bg-#fafafc p-3">
        <div class="text-xs text-gray-700 font-semibold">最近一次预览动作</div>
        <div class="mt-2 flex flex-wrap gap-2">
          <NTag v-for="item in integrationPreviewActions" :key="item" size="small" type="info" :bordered="false">
            {{ item }}
          </NTag>
        </div>
      </div>
    </NCard>

    <NCard :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">Coraza 引擎版本检查</div>
          <div class="mt-1 text-xs text-gray-500">
            用于发现 Coraza 引擎新版本并生成升级建议（需通过镜像发布流程升级）。
          </div>
        </div>
        <div class="flex gap-2">
          <NButton size="small" :loading="engineLoading" @click="handleRefreshEngineStatus">刷新状态</NButton>
          <NButton size="small" type="primary" :loading="engineChecking" @click="handleCheckEngine">
            检查上游版本
          </NButton>
        </div>
      </div>

      <NGrid cols="4" x-gap="12" y-gap="10" class="mt-4">
        <NGi>
          <div class="text-xs text-gray-500">当前版本</div>
          <div class="text-sm font-medium">
            {{ displayEngineValue(engineStatus?.currentVersion) }}
          </div>
        </NGi>
        <NGi>
          <div class="text-xs text-gray-500">最新版本</div>
          <div class="text-sm font-medium">
            {{ displayEngineValue(engineStatus?.latestVersion) }}
          </div>
        </NGi>
        <NGi>
          <div class="text-xs text-gray-500">可升级</div>
          <div class="text-sm font-medium">
            <NTag :type="engineStatus?.canUpgrade ? 'warning' : 'success'" :bordered="false">
              {{ engineStatus?.canUpgrade ? '是' : '否' }}
            </NTag>
          </div>
        </NGi>
        <NGi>
          <div class="text-xs text-gray-500">最近检查时间</div>
          <div class="text-sm font-medium">
            {{ displayEngineValue(engineStatus?.checkedAt) }}
          </div>
        </NGi>
      </NGrid>

      <NAlert v-if="engineUnavailable" type="warning" :show-icon="true" class="mt-4">
        当前引擎状态接口暂不可用，已切换为占位模式，请检查后端日志。
      </NAlert>
      <NAlert v-else-if="engineStatus?.message" type="info" :show-icon="true" class="mt-4">
        {{ engineStatus?.message }}
      </NAlert>
    </NCard>

    <NCard :bordered="false" class="rounded-8px shadow-sm">
      <SourceTabContent
        :source-query="sourceQuery"
        :source-columns="sourceColumns"
        :source-table="sourceTable"
        :source-loading="sourceLoading"
        :source-pagination="sourcePagination"
        :table-fixed-height="tableFixedHeight"
        :fetch-sources="fetchSources"
        :reset-source-query="resetSourceQuery"
        :handle-add-source="handleAddSource"
        :open-upload-modal="openUploadModal"
        :handle-source-page-change="handleSourcePageChange"
        :handle-source-page-size-change="handleSourcePageSizeChange"
      />
    </NCard>
  </div>
</template>
