<script setup lang="ts">
import { computed } from 'vue';
import SvgIcon from '@/components/custom/svg-icon.vue';
import PreservedBlockEditor from './PreservedBlockEditor.vue';
import type { ComplexSiteSummary, QuickSiteDraft } from '../quick-config-utils';
import type { PreservedCaddyBlock } from '../types';

const props = defineProps<{
  sites: QuickSiteDraft[];
  complexSites: ComplexSiteSummary[];
  preservedSiteBlocks?: PreservedCaddyBlock[];
}>();

const activeSiteId = defineModel<string | null>('activeSiteId', { required: true });

const emit = defineEmits<{
  (e: 'add'): void;
  (e: 'duplicate', id: string): void;
  (e: 'remove', id: string): void;
  (e: 'switch-raw'): void;
  (e: 'updatePreservedBlock', id: string, raw: string): void;
}>();

const activeSite = computed(() => props.sites.find(item => item.id === activeSiteId.value) || null);

const modeOptions = [
  { label: '反向代理', value: 'reverse_proxy' },
  { label: '静态站点', value: 'file_server' },
  { label: '重定向', value: 'redirect' }
];

const tlsOptions = [
  { label: '自动', value: 'auto' },
  { label: '关闭', value: 'off' },
  { label: 'internal', value: 'internal' }
];

function getSiteSecondaryText(site: QuickSiteDraft) {
  const domains = site.domains.filter(Boolean);
  return domains.length > 0 ? domains.join(', ') : '未配置域名/端口';
}

function domainLabel(domains: string[]) {
  const values = domains.filter(Boolean);
  if (values.length > 0 && values.every(value => /^:\d+$/.test(value))) return '端口';
  return '域名';
}

function quickErrors(site: QuickSiteDraft) {
  const errors: string[] = [];
  if (!site.name.trim()) errors.push('站点名称不能为空');
  if (site.domains.filter(Boolean).length === 0) errors.push('至少配置一个域名或端口');
  if (site.mode === 'reverse_proxy' && !site.upstream.trim()) errors.push('请填写代理目标地址');
  if (site.mode === 'file_server' && !site.root.trim()) errors.push('请填写站点根目录');
  if (site.mode === 'redirect' && !site.redirectTo.trim()) errors.push('请填写跳转地址');
  return errors;
}
</script>

<template>
  <div class="flex flex-col gap-4 lg:flex-row">
    <!-- 左栏：站点列表 + 插槽 -->
    <NCard size="small" class="quick-sidebar" content-style="padding: 0;">
      <div class="flex flex-col gap-4 p-4">
        <div>
          <div class="mb-3 flex items-center justify-between gap-2">
            <div>
              <div class="font-semibold">快速配置</div>
              <div class="mt-1 text-xs text-gray-500">只展示常用反代能力</div>
            </div>
            <NButton size="tiny" type="primary" @click="emit('add')">新建站点</NButton>
          </div>
        </div>

      <div v-if="props.sites.length === 0" class="h-full flex flex-col justify-center gap-3 py-6">
        <NEmpty :description="props.complexSites.length ? '当前没有可直接编辑的简单站点' : '还没有站点配置'" />
        <div class="flex flex-wrap justify-center gap-2">
          <NButton size="small" type="primary" @click="emit('add')">新建站点</NButton>
          <NButton size="small" @click="emit('switch-raw')">切换原始配置</NButton>
        </div>
      </div>

      <div v-else class="flex flex-col gap-2">
        <div
          v-for="site in props.sites"
          :key="site.id"
          class="cursor-pointer border rounded-lg px-3 py-2 transition"
          :class="activeSiteId === site.id ? 'border-primary-500 bg-primary-50' : 'border-gray-200 hover:border-primary-300'"
          @click="activeSiteId = site.id"
        >
          <div class="flex items-center justify-between gap-2">
            <div class="min-w-0">
              <div class="truncate font-medium">{{ site.name || '未命名站点' }}</div>
              <div class="truncate text-xs text-gray-500">{{ getSiteSecondaryText(site) }}</div>
            </div>
            <NTag size="small" :type="site.enabled ? 'success' : 'default'">
              {{ site.enabled ? '启用' : '停用' }}
            </NTag>
          </div>
          <div class="mt-2 flex items-center gap-2">
            <NButton size="tiny" @click.stop="emit('duplicate', site.id)">复制</NButton>
            <NButton size="tiny" type="error" @click.stop="emit('remove', site.id)">删除</NButton>
          </div>
        </div>
      </div>

      <!-- 额外侧栏内容（全局配置、上游等） -->
      <slot name="sidebar-extra" />
      </div>
    </NCard>

    <!-- 右栏：站点编辑 -->
    <NCard size="small" class="min-w-0 flex-1" content-style="padding: 0;">
      <div class="p-4">
        <NSpace vertical size="large">
        <NAlert v-if="props.complexSites.length" type="warning" :show-icon="true">
          检测到 {{ props.complexSites.length }} 个复杂站点。快速配置不会修改这些站点，请切换到原始配置维护高级规则。
        </NAlert>
        <div v-if="props.complexSites.length" class="-mt-2">
          <NButton text type="primary" @click="emit('switch-raw')">切换原始配置</NButton>
        </div>

        <NCard v-if="activeSite" size="small" :bordered="false">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="font-semibold">站点配置</div>
                <div class="mt-1 text-xs text-gray-500">聚焦域名、目标地址和 TLS 常用项</div>
              </div>
              <NTag size="small" :type="activeSite.enabled ? 'success' : 'default'">
                {{ activeSite.enabled ? '启用' : '停用' }}
              </NTag>
            </div>
          </template>

          <NForm label-placement="top">
            <div class="grid gap-4 lg:grid-cols-2">
              <NFormItem label="站点名称">
                <NInput v-model:value="activeSite.name" placeholder="例如：官网反代" />
              </NFormItem>
              <NFormItem label="启用状态">
                <NSwitch v-model:value="activeSite.enabled" />
              </NFormItem>
            </div>

            <NFormItem :label="domainLabel(activeSite.domains)">
              <NDynamicTags v-model:value="activeSite.domains" />
            </NFormItem>

            <div class="grid gap-4 lg:grid-cols-2">
              <NFormItem label="站点类型">
                <NSelect v-model:value="activeSite.mode" :options="modeOptions" />
              </NFormItem>
              <NFormItem label="TLS">
                <NSelect v-model:value="activeSite.tlsMode" :options="tlsOptions" />
              </NFormItem>
            </div>

            <NFormItem v-if="activeSite.mode === 'reverse_proxy'" label="代理目标">
              <NInput v-model:value="activeSite.upstream" placeholder="例如：127.0.0.1:8080 或 https://backend.internal" />
            </NFormItem>

            <template v-else-if="activeSite.mode === 'file_server'">
              <NFormItem label="站点根目录">
                <NInput v-model:value="activeSite.root" placeholder="例如：/srv/www/site" />
              </NFormItem>
              <NFormItem label="目录浏览">
                <NSwitch v-model:value="activeSite.browse" />
              </NFormItem>
            </template>

            <template v-else>
              <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_180px]">
                <NFormItem label="跳转地址">
                  <NInput v-model:value="activeSite.redirectTo" placeholder="例如：https://example.com" />
                </NFormItem>
                <NFormItem label="状态码">
                  <NInputNumber v-model:value="activeSite.redirectCode" :min="300" :max="399" />
                </NFormItem>
              </div>
            </template>
          </NForm>

          <NAlert v-if="quickErrors(activeSite).length" type="error" :show-icon="true" class="mt-2">
            {{ quickErrors(activeSite)[0] }}
          </NAlert>
        </NCard>

        <NEmpty v-else description="请选择或新建一个简单站点" />

        <!-- 只读保留站点（复杂 Caddyfile 块） -->
        <NCard v-if="props.preservedSiteBlocks?.length" size="small" :bordered="false">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <span class="font-semibold">只读保留站点</span>
                <NTag size="tiny" type="warning" :bordered="false" round>{{ props.preservedSiteBlocks.length }}</NTag>
              </div>
              <NButton size="small" @click="emit('switch-raw')">在原始配置中维护</NButton>
            </div>
          </template>

          <NCollapse>
            <NCollapseItem
              v-for="block in props.preservedSiteBlocks"
              :key="block.id"
              :name="block.id"
            >
              <template #header>
                <div class="flex items-center gap-2">
                  <NTag size="tiny" type="warning" :bordered="false" round>site</NTag>
                  <span class="text-sm font-medium">{{ block.title }}</span>
                </div>
              </template>
              <template #header-extra>
                <NTooltip trigger="hover">
                  <template #trigger>
                    <NIcon size="14" class="text-gray-400">
                      <SvgIcon icon="carbon:information" />
                    </NIcon>
                  </template>
                  {{ block.reason }}
                </NTooltip>
              </template>
              <PreservedBlockEditor :block="block" @update="(id, raw) => emit('updatePreservedBlock', id, raw)" />
            </NCollapseItem>
          </NCollapse>
        </NCard>
      </NSpace>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.quick-sidebar {
  width: 100%;
}

@media (min-width: 1024px) {
  .quick-sidebar {
    width: 45%;
    flex-shrink: 0;
  }
}
</style>
