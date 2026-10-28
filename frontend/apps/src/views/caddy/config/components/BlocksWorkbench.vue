<script lang="ts" setup>
import {
  Alert,
  Button,
  Collapse,
  CollapsePanel,
  Empty,
  Input,
  Popconfirm,
  Select,
  Space,
  Switch,
  Tag,
} from 'antdv-next';

import type { CaddyFormModel, HealthCheck, PreservedCaddyBlock } from '../types';
import type { ComplexSiteSummary, QuickSiteDraft } from '../quick-config-utils';
import { SITE_METRICS_RECENT_MINUTES } from '../../shared/composables/useSiteMetricsQuery';
import SiteAvatar from '../../shared/components/SiteAvatar.vue';
import SiteEditorPanel from './SiteEditorPanel.vue';

type PreservedItem = { block: PreservedCaddyBlock; index: number };

const props = defineProps<{
  formModel: CaddyFormModel;
  globalPreservedBlocks: PreservedItem[];
  preservedSiteBlocks: PreservedItem[];
  quickSiteDrafts: QuickSiteDraft[];
  complexSiteSummaries: ComplexSiteSummary[];
  activeQuickSiteId?: string;
  activeQuickSite: QuickSiteDraft | null | undefined;
  lbPolicyOptions: Array<{ label: string; value: string }>;
  quickModeOptions: Array<{ label: string; value: string }>;
  tlsModeOptions: Array<{ label: string; value: string }>;
  upstreamPoolOptions: Array<{ label: string; value: string }>;
  loadingSiteMetrics: boolean;
  siteMetricsLoaded: boolean;
  siteMetricsError: string;
  siteMetricsForDomains: (domains: string[] | undefined) => { count4xx: number; count5xx: number } | null | undefined;
  formatMetricCount: (value: number | undefined | null) => string;
  sitePrimaryHost: (domains: string[] | undefined) => string;
  blockKindLabel: (kind: PreservedCaddyBlock['kind']) => string;
  blockKindColor: (kind: PreservedCaddyBlock['kind']) => string;
  isUpstreamHealthEnabled: (index: number) => boolean;
  setUpstreamHealthEnabled: (index: number, enabled: boolean) => void;
  ensureUpstreamHealth: (index: number) => HealthCheck;
  isSiteHealthEnabled: (site: QuickSiteDraft) => boolean;
  setSiteHealthEnabled: (site: QuickSiteDraft, enabled: boolean) => void;
  ensureSiteHealth: (site: QuickSiteDraft) => HealthCheck;
}>();

const emit = defineEmits<{
  'update:activeQuickSiteId': [id: string];
  addUpstream: [];
  removeUpstream: [index: number];
  refreshMetrics: [];
  openDocker: [];
  addSite: [];
  openAccessLogs: [domains: string[], event: Event];
  duplicateSite: [site: QuickSiteDraft];
  removeSite: [id: string];
}>();

/** 卡片色条：停用=灰、出现 5xx=红、正常=绿，与目录卡片同一套语义 */
function siteAccent(site: QuickSiteDraft): string {
  if (!site.enabled) return 'hsl(var(--border))';
  if (metricValue(site.domains, '5xx') > 0) return 'hsl(var(--destructive))';
  return 'hsl(var(--success))';
}

/** 近窗指标原始值；未知按 0，避免模板里到处写可选链 */
function metricValue(domains: string[] | undefined, kind: '4xx' | '5xx'): number {
  const metrics = props.siteMetricsForDomains(domains);
  return (kind === '5xx' ? metrics?.count5xx : metrics?.count4xx) ?? 0;
}

/** 近窗指标展示值：加载中 …、失败 —、正常数字 */
function metricDisplay(domains: string[] | undefined, kind: '4xx' | '5xx'): string {
  if (props.loadingSiteMetrics && !props.siteMetricsLoaded) return '…';
  if (props.siteMetricsError) return '—';
  return props.formatMetricCount(metricValue(domains, kind));
}
</script>

<template>
  <div class="blocks-layout">
    <aside class="global-column">
      <Collapse
        :default-active-key="['global', 'global-preserved', 'upstreams']"
        class="config-collapse"
      >
        <CollapsePanel key="global" header="全局配置">
          <div class="panel-subtitle mb-3">Caddyfile 顶层选项，保存时会和右侧站点配置一起合并。</div>
          <Alert
            class="mb-3"
            type="info"
            show-icon
            message="自动 HTTPS、受信任代理、调试模式、ACME 邮箱、附加全局选项已由「全局设置」统一接管；在此编辑这些行，保存时会被设置值覆盖。其余指令（admin、order、servers 超时等）仍在此维护。"
          />
          <Input.TextArea
            v-model:value="props.formModel.global.raw"
            :auto-size="{ minRows: 14, maxRows: 22 }"
            class="code-textarea"
            placeholder="Caddy 全局选项"
          />
        </CollapsePanel>

        <CollapsePanel v-if="props.globalPreservedBlocks.length" key="global-preserved" header="Snippet / 保留块">
          <Alert
            class="mb-3"
            type="warning"
            show-icon
            message="snippet、全局片段和无法归类的配置只读保留原文，简单模式合并不会改写这些块。"
          />
          <Collapse
            :default-active-key="[]"
            class="inner-collapse"
          >
            <CollapsePanel
              v-for="item in props.globalPreservedBlocks"
              :key="item.block.id"
            >
              <template #header>
                <div class="preserved-title">
                  <Tag :color="props.blockKindColor(item.block.kind)">{{ props.blockKindLabel(item.block.kind) }}</Tag>
                  <span>{{ item.block.title }}</span>
                </div>
              </template>
              <div class="text-muted mb-2">原因：{{ item.block.reason || '无法安全结构化编辑' }}</div>
              <Input.TextArea
                :value="item.block.raw"
                :auto-size="{ minRows: 5, maxRows: 14 }"
                class="code-textarea"
                readonly
              />
            </CollapsePanel>
          </Collapse>
        </CollapsePanel>

        <CollapsePanel key="upstreams">
          <template #header>
            <div class="panel-header-line">
              <span>上游池</span>
              <Button size="small" @click.stop="emit('addUpstream')">新增</Button>
            </div>
          </template>
          <div class="panel-subtitle mb-3">
            复用后端目标、负载策略与健康检查。站点引用池名时，可在此统一配置探测；站点级健康检查优先于池级。
          </div>
          <Empty v-if="props.formModel.upstreams.length === 0" description="暂无上游池" />
          <div
            v-for="(upstream, index) in props.formModel.upstreams"
            :key="index"
            class="upstream-card"
          >
            <div class="upstream-row">
              <Input v-model:value="upstream.name" placeholder="池名称" />
              <Select
                v-model:value="upstream.lbPolicy"
                :options="props.lbPolicyOptions"
                placeholder="负载策略"
              />
              <Select
                v-model:value="upstream.targets"
                mode="tags"
                placeholder="目标地址"
              />
              <Button danger size="small" @click="emit('removeUpstream', index)">删除</Button>
            </div>
            <div class="health-row">
              <div class="health-toggle">
                <span class="health-label">健康检查</span>
                <Switch
                  :checked="props.isUpstreamHealthEnabled(index)"
                  checked-children="开"
                  un-checked-children="关"
                  @change="(checked: any) => props.setUpstreamHealthEnabled(index, Boolean(checked))"
                />
              </div>
              <template v-if="props.isUpstreamHealthEnabled(index)">
                <Input
                  :value="props.ensureUpstreamHealth(index).path"
                  placeholder="探测路径，如 /health"
                  @update:value="(value: string) => { props.ensureUpstreamHealth(index).path = value; }"
                />
                <Input
                  :value="props.ensureUpstreamHealth(index).interval"
                  placeholder="间隔，如 10s"
                  @update:value="(value: string) => { props.ensureUpstreamHealth(index).interval = value; }"
                />
                <Input
                  :value="props.ensureUpstreamHealth(index).timeout"
                  placeholder="超时，如 5s"
                  @update:value="(value: string) => { props.ensureUpstreamHealth(index).timeout = value; }"
                />
              </template>
            </div>
          </div>
        </CollapsePanel>
      </Collapse>
    </aside>

    <main class="sites-column">
      <Collapse
        :default-active-key="['sites', 'site-editor', 'site-preserved']"
        class="config-collapse"
      >
        <CollapsePanel key="sites">
          <template #header>
            <div class="panel-header-line">
              <span>站点</span>
              <Space @click.stop>
                <Button
                  size="small"
                  type="link"
                  :loading="props.loadingSiteMetrics"
                  @click="emit('refreshMetrics')"
                >
                  刷新指标
                </Button>
                <Button size="small" @click="emit('openDocker')">Docker 发现</Button>
                <Button size="small" type="primary" @click="emit('addSite')">新建站点</Button>
              </Space>
            </div>
          </template>
          <div class="panel-subtitle mb-3">
            右侧只维护站点能力；global 和 snippet 在左侧。卡片展示近 {{ SITE_METRICS_RECENT_MINUTES }} 分钟 5xx/4xx。
          </div>
          <Alert
            v-if="props.siteMetricsError"
            class="mb-3"
            type="warning"
            show-icon
            :message="`${props.siteMetricsError}（已降级显示，不影响配置编辑）`"
          />
          <div v-if="props.quickSiteDrafts.length === 0" class="sidebar-empty">
            <Empty description="暂无简单站点">
              <Space>
                <Button size="small" @click="emit('openDocker')">Docker 发现</Button>
                <Button size="small" type="primary" @click="emit('addSite')">新建站点</Button>
              </Space>
            </Empty>
          </div>
          <div v-else class="site-list">
            <button
              v-for="site in props.quickSiteDrafts"
              :key="site.id"
              type="button"
              class="site-item"
              :class="{ active: site.id === props.activeQuickSiteId }"
              @click="emit('update:activeQuickSiteId', site.id)"
            >
              <span class="site-item-accent" :style="{ background: siteAccent(site) }"></span>
              <div class="site-item-head">
                <SiteAvatar
                  :host="props.sitePrimaryHost(site.domains)"
                  :name="site.name"
                  :size="30"
                />
                <div class="site-title">
                  <span class="site-name">{{ site.name || '未命名站点' }}</span>
                  <span class="site-domain">{{ site.domains[0] || '未配置域名' }}</span>
                </div>
                <Tag class="site-state" :color="site.enabled ? 'green' : 'default'">
                  {{ site.enabled ? '启用' : '停用' }}
                </Tag>
              </div>
              <div class="site-metrics">
                <div class="site-metric">
                  <span class="site-metric-value" :class="{ danger: metricValue(site.domains, '5xx') > 0 }">
                    {{ metricDisplay(site.domains, '5xx') }}
                  </span>
                  <span class="meta-label">近窗 5xx</span>
                </div>
                <div class="site-metric-divider"></div>
                <div class="site-metric">
                  <span class="site-metric-value" :class="{ warn: metricValue(site.domains, '4xx') > 0 }">
                    {{ metricDisplay(site.domains, '4xx') }}
                  </span>
                  <span class="meta-label">近窗 4xx</span>
                </div>
                <Button
                  v-if="props.sitePrimaryHost(site.domains)"
                  size="small"
                  type="link"
                  class="site-log-link"
                  @click="(e: Event) => emit('openAccessLogs', site.domains, e)"
                >
                  查看日志
                </Button>
              </div>
            </button>
          </div>

          <Alert
            v-if="props.complexSiteSummaries.length"
            class="mt-3"
            show-icon
            type="warning"
            :message="`检测到 ${props.complexSiteSummaries.length} 个复杂站点，已在下方「保留站点 / 复杂站点」中只读展示，不可通过简单模式改写。`"
          />
        </CollapsePanel>

        <CollapsePanel key="site-editor">
          <template #header>
            <div class="panel-header-line">
              <span>站点配置</span>
              <Space v-if="props.activeQuickSite" @click.stop>
                <Button size="small" @click="emit('duplicateSite', props.activeQuickSite!)">复制</Button>
                <Popconfirm title="确认删除该站点？" @confirm="emit('removeSite', props.activeQuickSite!.id)">
                  <Button size="small" danger>删除</Button>
                </Popconfirm>
              </Space>
            </div>
          </template>
          <SiteEditorPanel
            :active-quick-site="props.activeQuickSite"
            :quick-mode-options="props.quickModeOptions"
            :tls-mode-options="props.tlsModeOptions"
            :lb-policy-options="props.lbPolicyOptions"
            :upstream-pool-options="props.upstreamPoolOptions"
            :is-site-health-enabled="props.isSiteHealthEnabled"
            :set-site-health-enabled="props.setSiteHealthEnabled"
            :ensure-site-health="props.ensureSiteHealth"
          />
        </CollapsePanel>

        <CollapsePanel
          v-if="props.preservedSiteBlocks.length || props.complexSiteSummaries.length"
          key="site-preserved"
          header="保留站点 / 复杂站点"
        >
          <Alert
            class="mb-3"
            type="warning"
            show-icon
            message="复杂站点与保留块只读展示原文与中文原因，简单模式合并不得改写保留块。"
          />

          <div v-if="props.complexSiteSummaries.length" class="complex-summary-list mb-3">
            <div
              v-for="item in props.complexSiteSummaries"
              :key="`complex-${item.id}`"
              class="complex-summary-item"
            >
              <div class="complex-summary-title">
                <span>{{ item.name || '未命名复杂站点' }}</span>
                <Tag color="orange">复杂站点</Tag>
              </div>
              <div class="text-muted mb-1">
                {{ item.domains.join(', ') || '未配置域名' }}
              </div>
              <div class="site-metrics complex-site-metrics mb-2">
                <div class="site-metric">
                  <span class="site-metric-value" :class="{ danger: metricValue(item.domains, '5xx') > 0 }">
                    {{ metricDisplay(item.domains, '5xx') }}
                  </span>
                  <span class="meta-label">近窗 5xx</span>
                </div>
                <div class="site-metric-divider"></div>
                <div class="site-metric">
                  <span class="site-metric-value" :class="{ warn: metricValue(item.domains, '4xx') > 0 }">
                    {{ metricDisplay(item.domains, '4xx') }}
                  </span>
                  <span class="meta-label">近窗 4xx</span>
                </div>
                <Button
                  v-if="props.sitePrimaryHost(item.domains)"
                  size="small"
                  type="link"
                  class="site-log-link"
                  @click="(e: Event) => emit('openAccessLogs', item.domains, e)"
                >
                  查看日志
                </Button>
              </div>
              <template v-if="item.reasons.length">
                <div class="text-muted mb-1">无法在简单模式编辑，原因：</div>
                <ul class="complex-reason-list">
                  <li v-for="(reason, rIndex) in item.reasons" :key="rIndex">
                    {{ reason }}
                  </li>
                </ul>
              </template>
              <div v-else class="text-muted">该站点含自定义片段，简单模式保存会丢失，已保持只读。</div>
            </div>
          </div>

          <Collapse
            v-if="props.preservedSiteBlocks.length"
            :default-active-key="[]"
            class="inner-collapse"
          >
            <CollapsePanel
              v-for="item in props.preservedSiteBlocks"
              :key="item.block.id"
            >
              <template #header>
                <div class="preserved-title">
                  <Tag color="orange">site</Tag>
                  <span>{{ item.block.title }}</span>
                </div>
              </template>
              <div v-if="item.block.reason" class="text-muted mb-2">原因：{{ item.block.reason }}</div>
              <Input.TextArea
                :value="item.block.raw"
                :auto-size="{ minRows: 5, maxRows: 14 }"
                class="code-textarea"
                readonly
              />
            </CollapsePanel>
          </Collapse>
        </CollapsePanel>
      </Collapse>
    </main>
  </div>
</template>

<style scoped>
.blocks-layout {
  display: grid;
  grid-template-columns: minmax(320px, 0.9fr) minmax(0, 1.45fr);
  gap: 16px;
  align-items: start;
}

.global-column,
.sites-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.config-collapse {
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  overflow: hidden;
  background: hsl(var(--card));
}

.config-collapse :deep(.ant-collapse-item) {
  border-bottom-color: hsl(var(--border));
}

.config-collapse :deep(.ant-collapse-header) {
  align-items: center;
  padding: 12px 16px;
  color: hsl(var(--foreground));
  font-weight: 600;
}

.config-collapse :deep(.ant-collapse-content-box) {
  padding: 14px 16px 16px;
}

.inner-collapse {
  border-color: hsl(var(--border));
  background: hsl(var(--muted) / 40%);
}

.inner-collapse :deep(.ant-collapse-header) {
  padding: 8px 12px;
  font-weight: 500;
}

.inner-collapse :deep(.ant-collapse-content-box) {
  padding: 10px 12px 12px;
}

.panel-header-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
}

.panel-subtitle {
  margin-top: 2px;
  color: hsl(var(--muted-foreground));
  font-size: 12px;
}

.preserved-title {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
}

.preserved-title span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-empty {
  padding: 18px 0;
}

.site-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}

.site-item {
  position: relative;
  width: 100%;
  padding: 12px;
  overflow: hidden;
  text-align: left;
  cursor: pointer;
  background: hsl(var(--muted) / 40%);
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    background 0.16s ease,
    transform 0.16s ease;
}

.site-item:hover {
  border-color: hsl(var(--primary) / 35%);
  box-shadow: 0 6px 20px hsl(var(--foreground) / 8%);
  transform: translateY(-1px);
}

.site-item.active {
  border-color: hsl(var(--primary));
  background: hsl(var(--primary) / 10%);
  box-shadow: 0 0 0 2px hsl(var(--primary) / 8%);
}

/* 顶部状态色条：正常=绿、出现 5xx=红、停用=灰 */
.site-item-accent {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
}

.site-item-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.site-title {
  display: flex;
  overflow: hidden;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  line-height: 1.35;
}

.site-state {
  flex-shrink: 0;
  margin-inline-end: 0;
}

.site-name,
.site-domain {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-name {
  font-size: 13px;
  font-weight: 600;
}

.site-domain {
  color: hsl(var(--muted-foreground));
  font-size: 12px;
}

.site-metrics {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 10px;
  padding: 8px 10px;
  background: hsl(var(--card) / 60%);
  border-radius: 8px;
}

.site-metric {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.site-metric-divider {
  align-self: stretch;
  width: 1px;
  background: hsl(var(--border));
}

.site-metric-value {
  font-size: 16px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.site-metric-value.danger {
  color: hsl(var(--destructive));
}

.site-metric-value.warn {
  color: hsl(var(--warning));
}

.site-log-link {
  height: auto;
  padding: 0 2px;
  font-size: 12px;
}

.complex-site-metrics {
  margin-top: 4px;
}

.meta-label {
  color: hsl(var(--muted-foreground) / 80%);
}

.text-muted {
  color: hsl(var(--muted-foreground));
  font-size: 12px;
}

.upstream-row {
  display: grid;
  grid-template-columns: minmax(110px, 0.8fr) minmax(120px, 0.7fr) minmax(160px, 1fr) auto;
  gap: 8px;
  margin-bottom: 8px;
}

.upstream-card {
  margin-bottom: 12px;
  padding: 12px;
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  background: hsl(var(--muted) / 40%);
}

.health-row {
  display: grid;
  grid-template-columns: auto minmax(120px, 1fr) minmax(90px, 0.7fr) minmax(90px, 0.7fr);
  gap: 8px;
  align-items: center;
  margin-top: 8px;
}

.health-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.health-label {
  color: hsl(var(--foreground) / 80%);
  font-size: 13px;
}

.complex-summary-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.complex-summary-item {
  padding: 12px;
  border: 1px solid hsl(var(--warning) / 40%);
  border-radius: 8px;
  background: hsl(var(--warning) / 12%);
}

.complex-summary-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
  font-weight: 600;
}

.complex-reason-list {
  margin: 0;
  padding-left: 18px;
  color: hsl(var(--muted-foreground));
  font-size: 12px;
}

.complex-reason-list li + li {
  margin-top: 2px;
}

.code-textarea {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 13px;
}

.mb-3 {
  margin-bottom: 12px;
}

.mb-2 {
  margin-bottom: 8px;
}

.mb-1 {
  margin-bottom: 4px;
}

.mt-3 {
  margin-top: 12px;
}

@media (max-width: 960px) {
  .blocks-layout {
    grid-template-columns: 1fr;
  }

  .upstream-row {
    grid-template-columns: 1fr;
  }

  .health-row {
    grid-template-columns: 1fr;
  }
}
</style>
