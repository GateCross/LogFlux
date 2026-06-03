<script setup lang="ts">
import { computed } from 'vue';
import { Icon } from '@iconify/vue';
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
    <Card size="small" class="quick-sidebar" body-style="padding: 0;">
      <div class="flex flex-col gap-4 p-4">
        <div>
          <div class="mb-3 flex items-center justify-between gap-2">
            <div>
              <div class="font-semibold">快速配置</div>
              <div class="mt-1 text-xs text-gray-500">只展示常用反代能力</div>
            </div>
            <Button size="tiny" type="primary" @click="emit('add')">新建站点</Button>
          </div>
        </div>

      <div v-if="props.sites.length === 0" class="h-full flex flex-col justify-center gap-3 py-6">
        <Empty :description="props.complexSites.length ? '当前没有可直接编辑的简单站点' : '还没有站点配置'" />
        <div class="flex flex-wrap justify-center gap-2">
          <Button size="small" type="primary" @click="emit('add')">新建站点</Button>
          <Button size="small" @click="emit('switch-raw')">切换原始配置</Button>
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
            <Tag size="small" :type="site.enabled ? 'success' : 'default'">
              {{ site.enabled ? '启用' : '停用' }}
            </Tag>
          </div>
          <div class="mt-2 flex items-center gap-2">
            <Button size="tiny" @click.stop="emit('duplicate', site.id)">复制</Button>
            <Button size="tiny" type="error" @click.stop="emit('remove', site.id)">删除</Button>
          </div>
        </div>
      </div>

      <!-- 额外侧栏内容（全局配置、上游等） -->
      <slot name="sidebar-extra" />
      </div>
    </Card>

    <!-- 右栏：站点编辑 -->
    <Card size="small" class="min-w-0 flex-1" body-style="padding: 0;">
      <div class="p-4">
        <Space vertical size="large">
        <Alert v-if="props.complexSites.length" type="warning" :show-icon="true">
          检测到 {{ props.complexSites.length }} 个复杂站点。快速配置不会修改这些站点，请切换到原始配置维护高级规则。
        </Alert>
        <div v-if="props.complexSites.length" class="-mt-2">
          <Button text type="primary" @click="emit('switch-raw')">切换原始配置</Button>
        </div>

        <Card v-if="activeSite" size="small" :bordered="false">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="font-semibold">站点配置</div>
                <div class="mt-1 text-xs text-gray-500">聚焦域名、目标地址和 TLS 常用项</div>
              </div>
              <Tag size="small" :type="activeSite.enabled ? 'success' : 'default'">
                {{ activeSite.enabled ? '启用' : '停用' }}
              </Tag>
            </div>
          </template>

          <Form label-placement="top">
            <div class="grid gap-4 lg:grid-cols-2">
              <Form.Item label="站点名称">
                <Input v-model:value="activeSite.name" placeholder="例如：官网反代" />
              </Form.Item>
              <Form.Item label="启用状态">
                <Switch v-model:value="activeSite.enabled" />
              </Form.Item>
            </div>

            <Form.Item :label="domainLabel(activeSite.domains)">
              <Select mode="tags" v-model:value="activeSite.domains" placeholder="输入后回车添加" />
            </Form.Item>

            <div class="grid gap-4 lg:grid-cols-2">
              <Form.Item label="站点类型">
                <Select v-model:value="activeSite.mode" :options="modeOptions" />
              </Form.Item>
              <Form.Item label="TLS">
                <Select v-model:value="activeSite.tlsMode" :options="tlsOptions" />
              </Form.Item>
            </div>

            <Form.Item v-if="activeSite.mode === 'reverse_proxy'" label="代理目标">
              <Input v-model:value="activeSite.upstream" placeholder="例如：127.0.0.1:8080 或 https://backend.internal" />
            </Form.Item>

            <template v-else-if="activeSite.mode === 'file_server'">
              <Form.Item label="站点根目录">
                <Input v-model:value="activeSite.root" placeholder="例如：/srv/www/site" />
              </Form.Item>
              <Form.Item label="目录浏览">
                <Switch v-model:value="activeSite.browse" />
              </Form.Item>
            </template>

            <template v-else>
              <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_180px]">
                <Form.Item label="跳转地址">
                  <Input v-model:value="activeSite.redirectTo" placeholder="例如：https://example.com" />
                </Form.Item>
                <Form.Item label="状态码">
                  <InputNumber v-model:value="activeSite.redirectCode" :min="300" :max="399" />
                </Form.Item>
              </div>
            </template>
          </Form>

          <Alert v-if="quickErrors(activeSite).length" type="error" :show-icon="true" class="mt-2">
            {{ quickErrors(activeSite)[0] }}
          </Alert>
        </Card>

        <Empty v-else description="请选择或新建一个简单站点" />

        <!-- 只读保留站点（复杂 Caddyfile 块） -->
        <Card v-if="props.preservedSiteBlocks?.length" size="small" :bordered="false">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <span class="font-semibold">只读保留站点</span>
                <Tag size="tiny" type="warning" :bordered="false" round>{{ props.preservedSiteBlocks.length }}</Tag>
              </div>
              <Button size="small" @click="emit('switch-raw')">在原始配置中维护</Button>
            </div>
          </template>

          <Collapse>
            <Collapse.Panel
              v-for="block in props.preservedSiteBlocks"
              :key="block.id"
              :name="block.id"
            >
              <template #header>
                <div class="flex items-center gap-2">
                  <Tag size="tiny" type="warning" :bordered="false" round>site</Tag>
                  <span class="text-sm font-medium">{{ block.title }}</span>
                </div>
              </template>
              <template #header-extra>
                <Tooltip trigger="hover">
                  <template #trigger>
                    <Icon icon="carbon:information" class="text-gray-400" />
                  </template>
                  {{ block.reason }}
                </Tooltip>
              </template>
              <PreservedBlockEditor :block="block" @update="(id, raw) => emit('updatePreservedBlock', id, raw)" />
            </Collapse.Panel>
          </Collapse>
        </Card>
      </Space>
      </div>
    </Card>
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
