<script setup lang="ts">
import { computed, toRefs, watchEffect } from 'vue';
import type { Site } from '../types';
import SiteRoutesEditor from './SiteRoutesEditor.vue';

const site = defineModel<Site | null>('site', { required: true });
const activeTab = defineModel<'basic' | 'routes' | 'advanced'>('tab', { default: 'basic' });
const props = defineProps<{ focusRouteId?: string | null }>();
const { focusRouteId } = toRefs(props);
const tlsOptions = [
  { label: 'auto', value: 'auto' },
  { label: 'off', value: 'off' },
  { label: 'internal', value: 'internal' },
  { label: 'manual', value: 'manual' }
];

const domainLabel = computed(() => {
  if (!site.value) return '域名';
  const values = site.value.domains.filter(Boolean);
  if (values.length > 0 && values.every(v => /^:\d+$/.test(v))) return '端口';
  return '域名';
});

watchEffect(() => {
  if (!site.value) return;
  if (!site.value.tls) site.value.tls = { mode: 'auto' };
  if (!site.value.imports) site.value.imports = [];
  if (!site.value.geoip2Vars) site.value.geoip2Vars = [];
  if (!site.value.encode) site.value.encode = [];
});

function invalidDomains(domains: string[]) {
  const re = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$/;
  const portOnlyRe = /^:\d+$/;
  return domains.filter(d => d && !(re.test(d) || portOnlyRe.test(d)));
}
</script>

<template>
  <NCard size="small" class="w-full" :bordered="false">
    <template #header>
      <div class="flex items-center justify-between">
        <span class="font-semibold">站点配置</span>
        <NTag v-if="site" size="small" :type="site.enabled ? 'success' : 'default'">
          {{ site.enabled ? '启用' : '停用' }}
        </NTag>
      </div>
    </template>
    <NEmpty v-if="!site" description="请选择一个站点" />
    <div v-else class="flex flex-col gap-4">
      <NTabs v-model:value="activeTab" type="line" size="small">
        <NTabPane name="basic" tab="基础">
          <NForm label-placement="left" label-width="80">
            <NFormItem label="名称">
              <NInput v-model:value="site.name" placeholder="站点名称" />
            </NFormItem>
            <NFormItem label="启用">
              <NSwitch v-model:value="site.enabled" />
            </NFormItem>
            <NFormItem :label="domainLabel">
              <NDynamicTags v-model:value="site.domains" />
            </NFormItem>
            <NFormItem label="TLS">
              <NSelect v-model:value="site.tls!.mode" :options="tlsOptions" class="w-40" />
            </NFormItem>
            <div v-if="site.tls?.mode === 'manual'" class="grid grid-cols-2 gap-2">
              <NInput v-model:value="site.tls!.certFile" placeholder="证书路径" />
              <NInput v-model:value="site.tls!.keyFile" placeholder="私钥路径" />
            </div>
          </NForm>
          <div v-if="!site.name" class="mt-2 text-xs text-red-500">站点名称不能为空</div>
          <div v-if="site.domains.length === 0" class="mt-1 text-xs text-red-500">至少配置一个域名</div>
          <div v-if="invalidDomains(site.domains).length" class="mt-1 text-xs text-red-500">
            域名格式不合法: {{ invalidDomains(site.domains).join(', ') }}
          </div>
        </NTabPane>
        <NTabPane name="routes" tab="路由">
          <SiteRoutesEditor v-model:routes="site.routes" :focus-route-id="focusRouteId" />
        </NTabPane>
        <NTabPane name="advanced" tab="高级">
          <div class="flex flex-col gap-4">
            <div>
              <div class="mb-1 text-sm font-medium">Import</div>
              <NDynamicTags v-model:value="site.imports" />
            </div>
            <div>
              <div class="mb-1 text-sm font-medium">GeoIP2 Vars</div>
              <NDynamicTags v-model:value="site.geoip2Vars" />
            </div>
            <div>
              <div class="mb-1 text-sm font-medium">Encode</div>
              <NDynamicTags v-model:value="site.encode" />
            </div>
          </div>
        </NTabPane>
      </NTabs>
    </div>
  </NCard>
</template>
