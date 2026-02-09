<template>
  <n-card size="small" class="w-full" :bordered="false">
    <template #header>
      <div class="flex items-center justify-between">
        <span class="font-semibold">站点配置</span>
        <n-tag v-if="site" size="small" :type="site.enabled ? 'success' : 'default'">
          {{ site.enabled ? '启用' : '停用' }}
        </n-tag>
      </div>
    </template>
    <n-empty v-if="!site" description="请选择一个站点" />
    <div v-else class="flex flex-col gap-4">
      <n-tabs v-model:value="activeTab" type="line" size="small">
        <n-tab-pane name="basic" tab="基础">
          <n-form label-placement="left" label-width="80">
            <n-form-item label="名称">
              <n-input v-model:value="site.name" placeholder="站点名称" />
            </n-form-item>
            <n-form-item label="启用">
              <n-switch v-model:value="site.enabled" />
            </n-form-item>
            <n-form-item :label="domainLabel">
              <n-dynamic-tags v-model:value="site.domains" />
            </n-form-item>
            <n-form-item label="TLS">
              <n-select v-model:value="site.tls!.mode" :options="tlsOptions" class="w-40" />
            </n-form-item>
            <div v-if="site.tls?.mode === 'manual'" class="grid grid-cols-2 gap-2">
              <n-input v-model:value="site.tls!.certFile" placeholder="证书路径" />
              <n-input v-model:value="site.tls!.keyFile" placeholder="私钥路径" />
            </div>
          </n-form>
          <div v-if="!site.name" class="text-xs text-red-500 mt-2">站点名称不能为空</div>
          <div v-if="site.domains.length === 0" class="text-xs text-red-500 mt-1">至少配置一个域名</div>
          <div v-if="invalidDomains(site.domains).length" class="text-xs text-red-500 mt-1">
            域名格式不合法: {{ invalidDomains(site.domains).join(', ') }}
          </div>
        </n-tab-pane>
        <n-tab-pane name="routes" tab="路由">
          <SiteRoutesEditor v-model:routes="site.routes" :focus-route-id="focusRouteId" />
        </n-tab-pane>
        <n-tab-pane name="advanced" tab="高级">
          <div class="flex flex-col gap-4">
            <div>
              <div class="text-sm font-medium mb-1">Import</div>
              <n-dynamic-tags v-model:value="site.imports" />
            </div>
            <div>
              <div class="text-sm font-medium mb-1">GeoIP2 Vars</div>
              <n-dynamic-tags v-model:value="site.geoip2Vars" />
            </div>
            <div>
              <div class="text-sm font-medium mb-1">Encode</div>
              <n-dynamic-tags v-model:value="site.encode" />
            </div>
          </div>
        </n-tab-pane>
      </n-tabs>
    </div>
  </n-card>
</template>

<script setup lang="ts">
import { watchEffect, computed } from 'vue';
import { toRefs } from 'vue';
import SiteRoutesEditor from './SiteRoutesEditor.vue';
import type { Site } from '../types';

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
