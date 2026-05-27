<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import { VueDraggable } from 'vue-draggable-plus';
import type { Handle, HandleType, HeaderRule, KeyValue, Route } from '../types';

const routes = defineModel<Route[]>('routes', { required: true });
const props = defineProps<{ focusRouteId?: string | null }>();

const handleOptions = [
  { label: 'reverse_proxy', value: 'reverse_proxy' },
  { label: 'file_server', value: 'file_server' },
  { label: 'respond', value: 'respond' },
  { label: 'redirect', value: 'redirect' },
  { label: 'header', value: 'header' },
  { label: 'rewrite', value: 'rewrite' }
];

const lbOptions = [
  { label: 'round_robin', value: 'round_robin' },
  { label: 'least_conn', value: 'least_conn' },
  { label: 'ip_hash', value: 'ip_hash' }
];

const headerOpOptions = [
  { label: 'set', value: 'set' },
  { label: 'add', value: 'add' },
  { label: 'delete', value: 'delete' }
];
const transportOptions = [
  { label: '默认', value: '' },
  { label: 'http', value: 'http' },
  { label: 'h2c', value: 'h2c' },
  { label: 'fastcgi', value: 'fastcgi' },
  { label: 'grpc', value: 'grpc' }
];

const expanded = ref<string[]>([]);
const methodAllowList = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];

function genId() {
  return (crypto as any).randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function createKeyValue(): KeyValue {
  return { key: '', value: '' };
}

function createHeaderRule(): HeaderRule {
  return { op: 'set', key: '', value: '' };
}

function invalidHosts(hosts: string[]) {
  const re = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$/;
  return hosts.filter(h => h && !re.test(h));
}

function invalidPaths(paths: string[]) {
  return paths.filter(p => p && !p.startsWith('/'));
}

function invalidMethods(methods: string[]) {
  return methods.filter(m => m && !methodAllowList.includes(m.toUpperCase()));
}

function routeSummary(route: Route) {
  const parts: string[] = [];
  if (route.match.host.length) parts.push(`Host:${route.match.host.length}`);
  if (route.match.path.length) parts.push(`Path:${route.match.path.length}`);
  if (route.match.method.length) parts.push(`Method:${route.match.method.length}`);
  const handlers = route.handles.map(h => h.type).filter(Boolean);
  const handlerText = handlers.length
    ? `Handlers:${handlers.slice(0, 3).join(',')}${handlers.length > 3 ? '…' : ''}`
    : 'Handlers:0';
  parts.push(handlerText);
  return parts.join(' · ');
}

function addRoute() {
  const id = genId();
  routes.value.push({
    id,
    name: '',
    enabled: true,
    match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
    logAppend: [],
    handles: []
  });
  if (!expanded.value.includes(id)) expanded.value.push(id);
}

function duplicateRoute(id: string) {
  const target = routes.value.find(r => r.id === id);
  if (!target) return;
  routes.value.push({
    ...JSON.parse(JSON.stringify(target)),
    id: genId(),
    name: `${target.name || '路由'}-copy`
  });
}

function removeRoute(id: string) {
  const idx = routes.value.findIndex(r => r.id === id);
  if (idx >= 0) routes.value.splice(idx, 1);
  expanded.value = expanded.value.filter(item => item !== id);
}

function toggleRoute(id: string) {
  if (expanded.value.includes(id)) {
    expanded.value = expanded.value.filter(item => item !== id);
  } else {
    expanded.value = [...expanded.value, id];
  }
}

function isExpanded(id: string) {
  return expanded.value.includes(id);
}

function addHandle(routeId: string) {
  const route = routes.value.find(r => r.id === routeId);
  if (!route) return;
  route.handles.push({
    id: genId(),
    type: 'reverse_proxy',
    enabled: true,
    upstream: '',
    lbPolicy: 'round_robin'
  });
}

function removeHandle(routeId: string, handleId: string) {
  const route = routes.value.find(r => r.id === routeId);
  if (!route) return;
  const idx = route.handles.findIndex(h => h.id === handleId);
  if (idx >= 0) route.handles.splice(idx, 1);
}

function handleTypeChange(handle: Handle, value: HandleType) {
  handle.type = value;
  if (value === 'reverse_proxy') {
    handle.upstream = handle.upstream ?? '';
    handle.lbPolicy = handle.lbPolicy ?? 'round_robin';
  }
  if (value === 'file_server') {
    handle.root = handle.root ?? '';
    handle.browse = handle.browse ?? false;
  }
  if (value === 'respond') {
    handle.status = handle.status ?? 200;
    handle.body = handle.body ?? '';
  }
  if (value === 'redirect') {
    handle.to = handle.to ?? '';
    handle.code = handle.code ?? 302;
  }
  if (value === 'header') {
    handle.rules = handle.rules ?? [];
  }
  if (value === 'rewrite') {
    handle.uri = handle.uri ?? '';
  }
}

watch(
  () => props.focusRouteId,
  async id => {
    if (!id) return;
    if (!expanded.value.includes(id)) {
      expanded.value = [...expanded.value, id];
    }
    await nextTick();
    const el = document.querySelector(`[data-route-id="${id}"]`);
    if (el instanceof HTMLElement) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }
);
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <div class="font-semibold">路由列表</div>
      <NButton size="small" type="primary" @click="addRoute">新增路由</NButton>
    </div>
    <NEmpty v-if="routes.length === 0" description="暂无路由" />
    <VueDraggable v-else v-model="routes" item-key="id" :animation="150" handle=".drag-handle">
      <div v-for="route in routes" :key="route.id" class="border border-gray-200 rounded p-3" :data-route-id="route.id">
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <icon-mdi-drag class="drag-handle cursor-move text-icon" />
            <span class="font-medium">{{ route.name || '未命名路由' }}</span>
          </div>
          <div class="flex items-center gap-2">
            <NSwitch v-model:value="route.enabled" size="small" />
            <NButton size="tiny" @click="toggleRoute(route.id)">
              {{ isExpanded(route.id) ? '收起' : '展开' }}
            </NButton>
          </div>
        </div>
        <div class="mt-1 text-xs text-gray-500">{{ routeSummary(route) }}</div>
        <NCollapseTransition :show="isExpanded(route.id)">
          <div class="mt-3 flex flex-col gap-3">
            <NInput v-model:value="route.name" placeholder="路由名称" />
            <div class="border border-gray-200 rounded-md p-3">
              <div class="mb-2 font-medium">Matchers</div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <div class="mb-1 text-xs text-gray-500">Host</div>
                  <NDynamicTags v-model:value="route.match.host" />
                  <div v-if="invalidHosts(route.match.host).length" class="mt-1 text-xs text-red-500">
                    Host 格式不合法: {{ invalidHosts(route.match.host).join(', ') }}
                  </div>
                </div>
                <div>
                  <div class="mb-1 text-xs text-gray-500">Path</div>
                  <NDynamicTags v-model:value="route.match.path" />
                  <div v-if="invalidPaths(route.match.path).length" class="mt-1 text-xs text-red-500">
                    Path 需以 / 开头: {{ invalidPaths(route.match.path).join(', ') }}
                  </div>
                </div>
                <div>
                  <div class="mb-1 text-xs text-gray-500">Method</div>
                  <NDynamicTags v-model:value="route.match.method" />
                  <div v-if="invalidMethods(route.match.method).length" class="mt-1 text-xs text-red-500">
                    Method 非法: {{ invalidMethods(route.match.method).join(', ') }}
                  </div>
                </div>
                <div>
                  <div class="mb-1 text-xs text-gray-500">Expression</div>
                  <NInput v-model:value="route.match.expression" placeholder="expression {expr}" />
                </div>
              </div>
              <div class="grid grid-cols-2 mt-3 gap-3">
                <div>
                  <div class="mb-1 text-xs text-gray-500">Header</div>
                  <NDynamicInput v-model:value="route.match.header" :on-create="createKeyValue">
                    <template #default="{ value }">
                      <div class="w-full flex gap-2">
                        <NInput v-model:value="value.key" placeholder="Key" />
                        <NInput v-model:value="value.value" placeholder="Value" />
                      </div>
                    </template>
                  </NDynamicInput>
                </div>
                <div>
                  <div class="mb-1 text-xs text-gray-500">Query</div>
                  <NDynamicInput v-model:value="route.match.query" :on-create="createKeyValue">
                    <template #default="{ value }">
                      <div class="w-full flex gap-2">
                        <NInput v-model:value="value.key" placeholder="Key" />
                        <NInput v-model:value="value.value" placeholder="Value" />
                      </div>
                    </template>
                  </NDynamicInput>
                </div>
              </div>
            </div>

            <div class="border border-gray-200 rounded-md p-3">
              <div class="mb-2 flex items-center justify-between">
                <div class="font-medium">Handlers</div>
                <NButton size="tiny" @click="addHandle(route.id)">新增 Handler</NButton>
              </div>
              <VueDraggable v-model="route.handles" item-key="id" :animation="150" handle=".handle-drag">
                <div v-for="handle in route.handles" :key="handle.id" class="border rounded p-2">
                  <div class="flex items-center gap-2">
                    <icon-mdi-drag class="handle-drag cursor-move text-icon" />
                    <NSelect
                      :value="handle.type"
                      :options="handleOptions"
                      size="small"
                      class="w-48"
                      @update:value="value => handleTypeChange(handle, value as HandleType)"
                    />
                    <NSwitch v-model:value="handle.enabled" size="small" />
                    <NButton size="tiny" type="error" @click="removeHandle(route.id, handle.id)">删除</NButton>
                  </div>

                  <div class="mt-2">
                    <template v-if="handle.type === 'reverse_proxy'">
                      <NInput
                        v-model:value="handle.upstream"
                        placeholder="上游名称或目标地址（host:port / https://host:port）"
                      />
                      <div v-if="!handle.upstream" class="mt-1 text-xs text-red-500">必须填写上游名称或目标地址</div>
                      <NSelect
                        v-model:value="handle.lbPolicy"
                        size="small"
                        class="mt-2 w-48"
                        :options="lbOptions"
                        placeholder="负载策略"
                      />
                      <NSelect
                        v-model:value="handle.transportProtocol"
                        size="small"
                        class="mt-2 w-48"
                        :options="transportOptions"
                        placeholder="传输协议"
                      />
                      <div class="mt-2">
                        <NSwitch v-model:value="handle.tlsInsecureSkipVerify" size="small" />
                        <span class="ml-2 text-sm">TLS 跳过校验</span>
                      </div>
                    </template>
                    <template v-else-if="handle.type === 'file_server'">
                      <NInput v-model:value="handle.root" placeholder="Root 路径" />
                      <div class="mt-2">
                        <NSwitch v-model:value="handle.browse" size="small" />
                        <span class="ml-2 text-sm">目录浏览</span>
                      </div>
                    </template>
                    <template v-else-if="handle.type === 'respond'">
                      <NInputNumber v-model:value="handle.status" placeholder="Status" :min="100" :max="599" />
                      <NInput v-model:value="handle.body" placeholder="Body" class="mt-2" />
                    </template>
                    <template v-else-if="handle.type === 'redirect'">
                      <NInput v-model:value="handle.to" placeholder="跳转地址" />
                      <NInputNumber
                        v-model:value="handle.code"
                        placeholder="状态码"
                        class="mt-2"
                        :min="300"
                        :max="399"
                      />
                    </template>
                    <template v-else-if="handle.type === 'header'">
                      <NDynamicInput v-model:value="handle.rules" :on-create="createHeaderRule">
                        <template #default="{ value }">
                          <div class="w-full flex gap-2">
                            <NSelect v-model:value="value.op" :options="headerOpOptions" size="small" class="w-24" />
                            <NInput v-model:value="value.key" placeholder="Key" />
                            <NInput v-model:value="value.value" placeholder="Value" />
                          </div>
                        </template>
                      </NDynamicInput>
                    </template>
                    <template v-else-if="handle.type === 'rewrite'">
                      <NInput v-model:value="handle.uri" placeholder="Rewrite URI" />
                    </template>
                  </div>
                </div>
              </VueDraggable>
            </div>

            <div class="border border-gray-200 rounded-md p-3">
              <div class="mb-2 font-medium">Log Append</div>
              <NDynamicInput v-model:value="route.logAppend" :on-create="createKeyValue">
                <template #default="{ value }">
                  <div class="w-full flex gap-2">
                    <NInput v-model:value="value.key" placeholder="Key" />
                    <NInput v-model:value="value.value" placeholder="Value" />
                  </div>
                </template>
              </NDynamicInput>
            </div>

            <div class="flex gap-2">
              <NButton size="tiny" @click="duplicateRoute(route.id)">复制路由</NButton>
              <NButton size="tiny" type="error" @click="removeRoute(route.id)">删除路由</NButton>
            </div>
          </div>
        </NCollapseTransition>
      </div>
    </VueDraggable>
  </div>
</template>
