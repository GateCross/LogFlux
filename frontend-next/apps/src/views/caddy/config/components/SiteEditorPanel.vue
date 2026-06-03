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
  <Card size="small" class="w-full" :bordered="false">
    <template #header>
      <div class="flex items-center justify-between">
        <span class="font-semibold">站点配置</span>
        <Tag v-if="site" size="small" :type="site.enabled ? 'success' : 'default'">
          {{ site.enabled ? '启用' : '停用' }}
        </Tag>
      </div>
    </template>
    <Empty v-if="!site" description="请选择一个站点" />
    <div v-else class="flex flex-col gap-4">
      <Tabs v-model:value="activeTab" type="line" size="small">
        <Tabs.TabPane name="basic" tab="基础">
          <Form label-placement="left" label-width="80">
            <Form.Item label="名称">
              <Input v-model:value="site.name" placeholder="站点名称" />
            </Form.Item>
            <Form.Item label="启用">
              <Switch v-model:value="site.enabled" />
            </Form.Item>
            <Form.Item :label="domainLabel">
              <Select mode="tags" v-model:value="site.domains" placeholder="输入后回车添加" />
            </Form.Item>
            <Form.Item label="TLS">
              <Select v-model:value="site.tls!.mode" :options="tlsOptions" class="w-40" />
            </Form.Item>
            <div v-if="site.tls?.mode === 'manual'" class="grid grid-cols-2 gap-2">
              <Input v-model:value="site.tls!.certFile" placeholder="证书路径" />
              <Input v-model:value="site.tls!.keyFile" placeholder="私钥路径" />
            </div>
          </Form>
          <div v-if="!site.name" class="mt-2 text-xs text-red-500">站点名称不能为空</div>
          <div v-if="site.domains.length === 0" class="mt-1 text-xs text-red-500">至少配置一个域名</div>
          <div v-if="invalidDomains(site.domains).length" class="mt-1 text-xs text-red-500">
            域名格式不合法: {{ invalidDomains(site.domains).join(', ') }}
          </div>
        </Tabs.TabPane>
        <Tabs.TabPane name="routes" tab="路由">
          <SiteRoutesEditor v-model:routes="site.routes" :focus-route-id="focusRouteId" />
        </Tabs.TabPane>
        <Tabs.TabPane name="advanced" tab="高级">
          <div class="flex flex-col gap-4">
            <div>
              <div class="mb-1 text-sm font-medium">Import</div>
              <Select mode="tags" v-model:value="site.imports" placeholder="输入后回车添加" />
            </div>
            <div>
              <div class="mb-1 text-sm font-medium">GeoIP2 Vars</div>
              <Select mode="tags" v-model:value="site.geoip2Vars" placeholder="输入后回车添加" />
            </div>
            <div>
              <div class="mb-1 text-sm font-medium">Encode</div>
              <Select mode="tags" v-model:value="site.encode" placeholder="输入后回车添加" />
            </div>
          </div>
        </Tabs.TabPane>
      </Tabs>
    </div>
  </Card>
</template>
