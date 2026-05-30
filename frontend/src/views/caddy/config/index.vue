<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from 'vue';
import { loader } from '@guolao/vue-monaco-editor';
import { useMessage } from 'naive-ui';
import type { CaddyBlockDraft } from './types';
import { buildCaddyfile, genId } from './caddy-config-utils';
import { parseCaddyfileToBlocks } from './caddy-config-blocks';
import { useCaddyServers } from './composables/useCaddyServers';
import { useCaddyConfigDraft } from './composables/useCaddyConfigDraft';
import { useCaddyPublishFlow } from './composables/useCaddyPublishFlow';
import { useCaddyConfigHistory } from './composables/useCaddyConfigHistory';
import { useSimpleWafBlock } from './composables/useSimpleWafBlock';
import SvgIcon from '@/components/custom/svg-icon.vue';
import ServerBlock from './components/ServerBlock.vue';
import SiteConfigBlock from './components/SiteConfigBlock.vue';
import UpstreamBlock from './components/UpstreamBlock.vue';
import GlobalSnippetBlock from './components/GlobalSnippetBlock.vue';
import WafBlock from './components/WafBlock.vue';
import RawConfigBlock from './components/RawConfigBlock.vue';
import PreviewPublishBlock from './components/PreviewPublishBlock.vue';
import HistoryBlock from './components/HistoryBlock.vue';

const VueMonacoDiffEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoDiffEditor)
);

loader.config({
  paths: { vs: 'https://registry.npmmirror.com/monaco-editor/0.44.0/files/min/vs' }
});

const message = useMessage();

// 1. 服务器管理
const servers = useCaddyServers({
  onAllServersRemoved: () => draft.resetState()
});

// 2. 配置草稿
const draft = useCaddyConfigDraft(servers.currentServerId);

// 3. 发布流程
const publish = useCaddyPublishFlow({
  currentServerId: servers.currentServerId,
  configContent: draft.configContent,
  mode: draft.mode,
  lastEditMode: draft.lastEditMode,
  structuredAvailable: draft.structuredAvailable,
  formModel: draft.formModel,
  preservedBlocks: draft.preservedBlocks,
  initialGlobalRaw: draft.initialGlobalRaw,
  mergedQuickFormModel: draft.mergedQuickFormModel,
  structuredReady: draft.structuredReady,
  syncQuickStateFromForm: draft.syncQuickStateFromForm,
  ensureBlocksFromRaw: draft.ensureBlocksFromRaw
});

// 4. 历史管理
const history = useCaddyConfigHistory({
  currentServerId: servers.currentServerId,
  formattedConfigContent: draft.formattedConfigContent,
  loadConfig: draft.loadConfig
});

// 5. WAF 管理
const waf = useSimpleWafBlock({
  currentServerId: servers.currentServerId,
  loadConfig: draft.loadConfig
});

// 全局配置对比弹窗
const showGlobalCompareModal = ref(false);
const showGlobalDiffOnly = ref(false);
const diffLeftRef = ref<HTMLElement | null>(null);
const diffRightRef = ref<HTMLElement | null>(null);
let diffSyncing = false;

function syncDiffScroll(side: 'left' | 'right') {
  if (diffSyncing) return;
  const source = side === 'left' ? diffLeftRef.value : diffRightRef.value;
  const target = side === 'left' ? diffRightRef.value : diffLeftRef.value;
  if (!source || !target) return;
  diffSyncing = true;
  target.scrollTop = source.scrollTop;
  target.scrollLeft = source.scrollLeft;
  requestAnimationFrame(() => { diffSyncing = false; });
}

const globalDiffRowsFiltered = computed(() => {
  if (!showGlobalDiffOnly.value) return draft.globalDiffRows.value;
  return draft.globalDiffRows.value.filter(row => row.type !== 'same');
});

// 只读保留站点（kind='site' 的 preservedBlocks）
const preservedSiteBlocks = computed(() =>
  draft.preservedBlocks.value.filter(b => b.kind === 'site')
);

// 从原始配置重新解析
function handleReparse() {
  publish.confirmOverwriteStructured('从原始配置解析', () => {
    if (!draft.configContent.value.trim()) {
      message.error('原始配置为空，无法解析');
      return;
    }
    const parsed = parseCaddyfileToBlocks(draft.configContent.value);
    if (parsed.sites.length === 0 && !parsed.global?.raw && parsed.preservedBlocks.length === 0) {
      message.error('未解析到可用结构化配置');
      return;
    }
    draft.applyDraft(parsed, true);
  });
}

// 应用默认模板
function applyPreset() {
  publish.confirmOverwriteStructured('应用默认模板', () => {
    draft.structuredAvailable.value = true;
    draft.formModel.value.schemaVersion = 1;
    draft.formModel.value.upstreams = [];
    const siteId = genId();
    draft.formModel.value.sites = [
      {
        id: siteId,
        name: '默认站点',
        enabled: true,
        domains: ['example.com'],
        imports: [],
        geoip2Vars: [],
        encode: [],
        tls: { mode: 'auto' },
        routes: [
          {
            id: genId(),
            name: '默认路由',
            enabled: true,
            match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
            logAppend: [],
            handles: [
              {
                id: genId(),
                type: 'reverse_proxy',
                enabled: true,
                upstream: 'localhost:8080',
                lbPolicy: 'round_robin',
                tlsInsecureSkipVerify: false
              }
            ]
          }
        ]
      }
    ];
    draft.initialGlobalRaw.value = draft.formModel.value.global?.raw ?? '';
    draft.syncQuickStateFromForm(draft.formModel.value);
    draft.activeQuickSiteId.value = siteId;
    draft.lastEditMode.value = 'blocks';
    draft.mode.value = 'blocks';
  });
}

// 更多操作菜单
const moreOptions = computed(() => [
  { label: '更多设置', key: 'settings' },
  { type: 'divider', key: 'divider-1' },
  { label: '添加服务器', key: 'server:add' },
  { label: '编辑当前服务器', key: 'server:edit', disabled: !servers.currentServerId.value },
  { label: '删除当前服务器', key: 'server:delete', disabled: !servers.currentServerId.value },
  { type: 'divider', key: 'divider-2' },
  { label: '查看历史版本', key: 'history', disabled: !servers.currentServerId.value },
  { label: '应用默认模板', key: 'preset' },
  { label: '从原始配置解析', key: 'import-raw' }
]);

const showSettingsDrawer = ref(false);

function handleMoreAction(key: string) {
  if (key === 'settings') { showSettingsDrawer.value = true; return; }
  if (key === 'server:add') { servers.openAddServerModal(); return; }
  if (key === 'server:edit') { servers.openEditServerModal(); return; }
  if (key === 'server:delete') { servers.handleDeleteServer(); return; }
  if (key === 'history') { history.openHistoryModal(); return; }
  if (key === 'preset') { applyPreset(); return; }
  if (key === 'import-raw') { handleReparse(); }
}

// 模式选项
const pageModeOptions = [
  { label: '分块配置', value: 'blocks' },
  { label: '防火墙', value: 'waf' },
  { label: '原始配置', value: 'raw' },
  { label: '预览', value: 'preview' }
] as const;

const pageModeSummary = computed(() => {
  if (draft.mode.value === 'blocks') return '只编辑常用站点能力，复杂配置自动保留。';
  if (draft.mode.value === 'waf') return '配置 Coraza / OWASP CRS 的常用开关。';
  if (draft.mode.value === 'raw') return '直接维护完整 Caddyfile，适合高级规则。';
  return draft.lastEditMode.value === 'raw' ? '展示当前原始配置内容。' : '展示当前快速配置生成结果。';
});

// 监听服务器切换
watch(servers.currentServerId, () => {
  if (servers.currentServerId.value) {
    draft.loadConfig();
    return;
  }
  draft.resetState();
});

onMounted(() => {
  servers.getServers();
});
</script>

<template>
  <div class="h-full flex flex-col overflow-hidden">
    <NCard
      class="h-full card-wrapper"
      :content-style="{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }"
    >
      <template #header>
        <div class="caddy-toolbar">
          <ServerBlock
            :servers="servers.servers.value"
            :server-options="servers.serverOptions.value"
            :current-server-id="servers.currentServerId.value"
            @update:current-server-id="servers.currentServerId.value = $event"
            @add="servers.openAddServerModal"
            @edit="servers.openEditServerModal"
            @delete="servers.handleDeleteServer"
          />
          <div class="caddy-toolbar-actions">
            <NButton
              v-if="draft.mode.value === 'blocks'"
              type="primary"
              size="small"
              :loading="publish.saving.value"
              :disabled="!servers.currentServerId.value"
              @click="publish.saveBlocksConfig"
            >
              保存快速配置
            </NButton>
            <NButton
              v-else-if="draft.mode.value === 'raw'"
              type="primary"
              size="small"
              :loading="publish.saving.value"
              :disabled="!servers.currentServerId.value"
              @click="publish.saveRawConfig"
            >
              保存原始配置
            </NButton>
            <NTag v-else-if="draft.mode.value === 'waf'" size="small" type="warning" :bordered="false">防火墙设置</NTag>
            <NTag v-else size="small" type="info" :bordered="false">预览模式</NTag>

            <NDropdown :options="moreOptions" @select="handleMoreAction">
              <NButton size="small" secondary>
                <div class="flex items-center gap-1">
                  <span>更多</span>
                  <SvgIcon icon="carbon:chevron-down" class="caddy-icon" />
                </div>
              </NButton>
            </NDropdown>
          </div>
        </div>
      </template>

      <div class="min-h-0 flex flex-col flex-1 gap-4 overflow-auto">
        <!-- 模式切换栏 -->
        <div class="caddy-mode-strip">
          <NRadioGroup :value="draft.mode.value" size="small" @update:value="publish.handleModeChange">
            <NRadioButton v-for="option in pageModeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </NRadioButton>
          </NRadioGroup>
          <div class="text-xs text-gray-500">{{ pageModeSummary }}</div>
        </div>

        <!-- 空状态 -->
        <div v-if="servers.servers.value.length === 0" class="h-full flex flex-col items-center justify-center p-8 text-gray-400">
          <div class="text-lg">未找到 Caddy 服务器</div>
          <div class="mt-2 text-sm">先添加一个服务器，再开始管理配置。</div>
          <NButton class="mt-4" type="primary" @click="servers.openAddServerModal">添加服务器</NButton>
        </div>

        <!-- 主内容区 -->
        <NSpin v-else :show="draft.loading.value" class="min-h-0 flex-1" content-class="h-full min-h-0">
          <NAlert
            v-if="draft.mode.value === 'blocks' && draft.quickValidationErrors.value?.length"
            type="error"
            :show-icon="true"
            class="mb-3"
          >
            {{ draft.quickValidationErrors.value[0] }}
          </NAlert>

          <!-- 分块配置 -->
          <SiteConfigBlock
            v-if="draft.mode.value === 'blocks'"
            v-model:active-site-id="draft.activeQuickSiteId.value"
            :sites="draft.quickSiteDrafts.value"
            :complex-sites="draft.complexSiteSummaries.value"
            :preserved-site-blocks="preservedSiteBlocks"
            @add="draft.addQuickSite"
            @duplicate="draft.duplicateQuickSite"
            @remove="draft.removeQuickSite"
            @switch-raw="publish.handleModeChange('raw')"
            @update-preserved-block="(id, raw) => draft.updatePreservedBlock(id, raw)"
          >
            <template #sidebar-extra>
              <GlobalSnippetBlock
                v-model:global-raw="draft.formModel.value.global.raw"
                :preserved-blocks="draft.preservedBlocks.value"
                :read-only="false"
                :initial-global-raw="draft.initialGlobalRaw.value"
                @compare="showGlobalCompareModal = true"
                @restore="draft.formModel.value.global.raw = draft.initialGlobalRaw.value ?? ''"
                @update-preserved-block="(id, raw) => draft.updatePreservedBlock(id, raw)"
              />
              <UpstreamBlock :upstreams="draft.formModel.value.upstreams" />
            </template>
          </SiteConfigBlock>

          <!-- 防火墙 -->
          <WafBlock
            v-else-if="draft.mode.value === 'waf'"
            :waf="waf"
          />

          <!-- 原始配置 -->
          <RawConfigBlock
            v-else-if="draft.mode.value === 'raw'"
            v-model="draft.configContent.value"
            @reparse="handleReparse"
          />

          <!-- 预览 -->
          <PreviewPublishBlock
            v-else
            v-model:save-preview="publish.savePreview.value"
            :config-content="draft.formattedConfigContent.value"
            :saving="publish.saving.value"
            @confirm="publish.confirmSavePreview"
            @close="publish.closeSavePreview"
          />
        </NSpin>
      </div>
    </NCard>

    <!-- 设置抽屉 -->
    <NDrawer v-model:show="showSettingsDrawer" :width="560" placement="right">
      <NDrawerContent title="更多设置" closable>
        <NSpace vertical size="large">
          <NCard size="small" :bordered="false">
            <template #header>配置工具</template>
            <div class="flex flex-wrap gap-2">
              <NButton size="small" @click="applyPreset">应用默认模板</NButton>
              <NButton size="small" @click="handleReparse">从原始配置解析</NButton>
              <NButton size="small" :disabled="!servers.currentServerId.value" @click="history.openHistoryModal">查看历史版本</NButton>
            </div>
          </NCard>
        </NSpace>
      </NDrawerContent>
    </NDrawer>

    <!-- 服务器管理弹窗 -->
    <NModal
      v-model:show="servers.showServerModal.value"
      preset="card"
      :title="servers.serverModalType.value === 'add' ? '添加服务器' : '编辑服务器'"
      class="w-500px"
    >
      <NForm label-placement="left" label-width="80">
        <NFormItem label="名称" path="name">
          <NInput v-model:value="servers.serverFormModel.value.name" placeholder="服务器名称" />
        </NFormItem>
        <NFormItem label="地址" path="url">
          <NInput v-model:value="servers.serverFormModel.value.url" placeholder="http://localhost:2019" />
        </NFormItem>
        <NFormItem label="类型" path="type">
          <NRadioGroup v-model:value="servers.serverFormModel.value.type">
            <NRadioButton value="local">本地</NRadioButton>
            <NRadioButton value="remote">远程</NRadioButton>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="servers.serverFormModel.value.type === 'remote'" label="凭证" path="token">
          <NInput v-model:value="servers.serverFormModel.value.token" placeholder="可选认证凭证" />
        </NFormItem>
        <div class="flex justify-end gap-2">
          <NButton @click="servers.showServerModal.value = false">取消</NButton>
          <NButton type="primary" @click="servers.handleSaveServer">保存</NButton>
        </div>
      </NForm>
    </NModal>

    <!-- 全局配置对比弹窗 -->
    <NModal v-model:show="showGlobalCompareModal" preset="card" title="全局配置对比" class="max-w-4xl w-[90vw]">
      <div class="diff-head">
        <div>已保存</div>
        <div class="flex items-center justify-between">
          <span>当前</span>
          <NSwitch v-model:value="showGlobalDiffOnly" size="small">
            <template #checked>仅差异</template>
            <template #unchecked>全部</template>
          </NSwitch>
        </div>
      </div>
      <div class="diff-body diff-two">
        <div ref="diffLeftRef" class="diff-column" @scroll="syncDiffScroll('left')">
          <div v-for="row in globalDiffRowsFiltered" :key="row.key" class="diff-line-row" :class="row.type">
            <span class="diff-no">{{ row.leftNo ?? '' }}</span>
            <span class="diff-line">{{ row.left !== null ? row.left : '' }}</span>
          </div>
        </div>
        <div ref="diffRightRef" class="diff-column" @scroll="syncDiffScroll('right')">
          <div v-for="row in globalDiffRowsFiltered" :key="row.key" class="diff-line-row" :class="row.type">
            <span class="diff-no">{{ row.rightNo ?? '' }}</span>
            <span class="diff-line">{{ row.right !== null ? row.right : '' }}</span>
          </div>
        </div>
      </div>
    </NModal>

    <!-- 历史管理组件 -->
    <HistoryBlock
      :history="history"
      :history-compare-right="draft.formattedConfigContent.value"
    />
  </div>
</template>

<style scoped>
:deep(.n-card__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
}
:deep(.n-spin-content) {
  height: 100%;
}

:deep(.monaco-editor-overlay) {
  z-index: 1000 !important;
}

.caddy-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.caddy-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.caddy-mode-strip {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 12px 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}

.caddy-icon {
  display: inline-block;
  font-size: 16px;
  line-height: 1;
  vertical-align: middle;
}

@media (max-width: 900px) {
  .caddy-toolbar {
    align-items: stretch;
  }
  .caddy-toolbar-actions {
    justify-content: flex-start;
    flex-wrap: wrap;
  }
}

.diff-head {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  font-size: 12px;
  color: #64748b;
  margin-bottom: 8px;
}

.diff-body {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  overflow: hidden;
  max-height: 60vh;
  background: #ffffff;
}

.diff-two {
  display: grid;
  grid-template-columns: 1fr 1fr;
}

.diff-column {
  overflow: auto;
  max-height: 60vh;
}

.diff-column + .diff-column {
  border-left: 1px solid #e2e8f0;
}

.diff-line-row {
  display: grid;
  grid-template-columns: 32px 1fr;
  gap: 8px;
  padding: 6px 10px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  white-space: pre-wrap;
}

.diff-line-row.added {
  background: #ecfdf3;
}

.diff-line-row.removed {
  background: #fef2f2;
}

.diff-line-row.changed {
  background: #fff7ed;
}

.diff-line {
  display: block;
  overflow-wrap: anywhere;
}

.diff-no {
  color: #94a3b8;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
</style>
