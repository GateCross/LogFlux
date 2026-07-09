<script lang="ts" setup>
/**
 * 服务目录页：复用 MVP 站点卡片 + probe + metrics + WAF/配置入口。
 * 数据源仅既有 GET /caddy/server、GET /caddy/server/status、
 * GET /caddy/server/:id/config、POST /caddy/logs/site-metrics；
 * 不建平行 discovery DB，不自动 /load。
 */
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useDebounceFn } from '@vueuse/core';

import { Page } from '@vben/common-ui';

import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Input,
  message,
  Row,
  Select,
  Space,
  Spin,
  Tag,
  Tooltip,
} from 'ant-design-vue';

import {
  getCaddyConfigApi,
  getCaddyServerListApi,
  getSiteMetricsApi,
  type CaddyServerApi,
  type SiteMetricsItem,
} from '#/api/caddy/server';

import {
  SITE_METRICS_HOST_CHUNK,
  SITE_METRICS_WINDOW_MINUTES,
  apiErrorMessage,
  buildAccessLogQuery,
  buildCatalogSitesFromConfigPayload,
  buildConfigWorkbenchQuery,
  buildSiteMetricsMap,
  catalogModeLabel,
  collectCatalogHosts,
  formatMetricCount,
  metricCountColor,
  siteMetricsForHost,
  sitePrimaryHost,
  type CatalogSiteCard,
  type SiteMetricsMap,
} from './service-catalog-utils';
import {
  formatLatency,
  statusErrorSummary,
  statusLabel,
  statusTagColor,
} from '../shared/caddy-server-status';
import { useCaddyServerStatus } from '../shared/use-caddy-server-status';

defineOptions({ name: 'CaddyCatalog' });

const router = useRouter();

type CaddyServer = Record<string, any>;

const servers = ref<CaddyServer[]>([]);
const selectedServerId = ref<number>();
const loadingServers = ref(false);
const loadingConfig = ref(false);

const siteCards = ref<CatalogSiteCard[]>([]);
const keyword = ref('');
const kindFilter = ref<'all' | 'simple' | 'complex'>('all');

/** 近窗指标：失败降级，不阻塞目录浏览 */
const siteMetricsMap = ref<SiteMetricsMap>(new Map());
const loadingSiteMetrics = ref(false);
const siteMetricsError = ref('');
const siteMetricsLoaded = ref(false);

const serverOptions = computed(() =>
  servers.value.map((server) => ({
    label: serverLabel(server),
    value: Number(server.id),
  })),
);

function serverLabel(server: CaddyServer) {
  return server.name ?? server.host ?? server.url ?? `Server #${server.id}`;
}

/** 服务目录：完整节点状态总览（手动探测，不 thrash） */
const {
  loadingServerStatus,
  serverStatusError,
  serverStatusLoaded,
  lastProbedAt,
  selectedServerStatus,
  serverStatusRows,
  onlineCount,
  offlineCount,
  pruneByServers,
  handleRefreshServerStatus,
} = useCaddyServerStatus({
  servers,
  selectedServerId,
  labelOf: serverLabel,
});

const filteredSites = computed(() => {
  const q = keyword.value.trim().toLowerCase();
  return siteCards.value.filter((site) => {
    if (kindFilter.value !== 'all' && site.kind !== kindFilter.value) return false;
    if (!q) return true;
    const hay = [site.name, site.primaryHost, ...(site.domains ?? []), site.upstream ?? '']
      .join(' ')
      .toLowerCase();
    return hay.includes(q);
  });
});

const summary = computed(() => {
  const total = siteCards.value.length;
  const simple = siteCards.value.filter((s) => s.kind === 'simple').length;
  const complex = total - simple;
  return {
    total,
    simple,
    complex,
    online: onlineCount.value,
    offline: offlineCount.value,
  };
});

async function fetchServers() {
  loadingServers.value = true;
  try {
    servers.value = await getCaddyServerListApi();
    if (servers.value.length > 0 && !selectedServerId.value) {
      selectedServerId.value = Number(servers.value[0]?.id);
    }
    pruneByServers(servers.value);
    if (servers.value.length === 0) {
      siteCards.value = [];
      siteMetricsMap.value = new Map();
    }
  } catch (error) {
    message.error(apiErrorMessage(error, '获取服务器列表失败'));
  } finally {
    loadingServers.value = false;
  }
}

async function fetchConfigSites() {
  if (!selectedServerId.value) {
    siteCards.value = [];
    return;
  }
  loadingConfig.value = true;
  siteMetricsMap.value = new Map();
  siteMetricsError.value = '';
  siteMetricsLoaded.value = false;
  try {
    const data = await getCaddyConfigApi(selectedServerId.value);
    siteCards.value = buildCatalogSitesFromConfigPayload(data as CaddyServerApi.CaddyConfig);
    debouncedFetchSiteMetrics();
  } catch (error) {
    siteCards.value = [];
    message.error(apiErrorMessage(error, '获取站点配置失败'));
  } finally {
    loadingConfig.value = false;
  }
}

async function fetchSiteMetrics() {
  if (loadingSiteMetrics.value) return;
  const hosts = collectCatalogHosts(siteCards.value);
  if (hosts.length === 0) {
    siteMetricsMap.value = new Map();
    siteMetricsError.value = '';
    siteMetricsLoaded.value = true;
    return;
  }

  loadingSiteMetrics.value = true;
  siteMetricsError.value = '';
  try {
    const items: SiteMetricsItem[] = [];
    for (let i = 0; i < hosts.length; i += SITE_METRICS_HOST_CHUNK) {
      const chunk = hosts.slice(i, i + SITE_METRICS_HOST_CHUNK);
      const list = await getSiteMetricsApi({
        hosts: chunk,
        windowMinutes: SITE_METRICS_WINDOW_MINUTES,
      });
      items.push(...((list as SiteMetricsItem[]) ?? []));
    }
    siteMetricsMap.value = buildSiteMetricsMap(items, hosts);
    siteMetricsLoaded.value = true;
  } catch (error) {
    siteMetricsError.value = apiErrorMessage(error, '获取站点近窗指标失败');
    siteMetricsMap.value = new Map();
    siteMetricsLoaded.value = true;
  } finally {
    loadingSiteMetrics.value = false;
  }
}

const debouncedFetchSiteMetrics = useDebounceFn(fetchSiteMetrics, 600);

function handleRefreshSiteMetrics() {
  debouncedFetchSiteMetrics();
}

function handleServerChange(value: unknown) {
  const next = Number(value);
  if (!Number.isFinite(next)) return;
  selectedServerId.value = next;
}

function selectServerFromStatus(id: number) {
  if (!Number.isFinite(id)) return;
  selectedServerId.value = id;
}

/** 打开配置工作台（同源配置，不复制数据源） */
function openConfigWorkbench(mode?: 'blocks' | 'waf' | 'raw' | 'preview') {
  router.push({
    name: 'CaddyConfig',
    query: buildConfigWorkbenchQuery({
      serverId: selectedServerId.value,
      mode,
    }),
  });
}

function openWaf() {
  openConfigWorkbench('waf');
}

function openAccessLogs(domains: string[] | undefined, event?: Event) {
  event?.stopPropagation();
  event?.preventDefault();
  const query = buildAccessLogQuery(domains);
  if (!query.host) {
    message.warning('该站点未配置可用域名，无法过滤访问日志');
    return;
  }
  router.push({
    name: 'CaddyLog',
    query,
  });
}

function openSiteInWorkbench(site: CatalogSiteCard) {
  // 复杂站点同样进配置工作台查看；不在目录页改写配置
  openConfigWorkbench('blocks');
  void site;
}

function metricDisplay(site: CatalogSiteCard, kind: '4xx' | '5xx') {
  if (loadingSiteMetrics.value && !siteMetricsLoaded.value) return '…';
  if (siteMetricsError.value) return '—';
  const metrics = siteMetricsForHost(siteMetricsMap.value, site.domains);
  return formatMetricCount(kind === '5xx' ? metrics?.count5xx : metrics?.count4xx);
}

watch(selectedServerId, (id) => {
  if (id) {
    fetchConfigSites();
  } else {
    siteCards.value = [];
  }
});

onMounted(async () => {
  // 选中节点后由 watch(selectedServerId) 拉取站点；避免与 watch 重复请求
  await fetchServers();
});
</script>

<template>
  <Page auto-content-height>
    <div class="service-catalog-page">
      <Card :bordered="false" class="catalog-shell">
        <template #title>
          <div class="catalog-title-row">
            <span>服务目录</span>
            <span class="catalog-subtitle">
              复用节点探测、站点近窗指标与配置工作台入口（不复制数据源，不自动热加载）
            </span>
          </div>
        </template>
        <template #extra>
          <Space wrap>
            <Select
              :value="selectedServerId"
              :options="serverOptions"
              :loading="loadingServers"
              class="server-select"
              placeholder="选择 Caddy 节点"
              @change="handleServerChange"
            />
            <Tooltip
              v-if="selectedServerStatus"
              :title="
                statusErrorSummary(selectedServerStatus) ||
                `延迟 ${formatLatency(selectedServerStatus.latencyMs)} · ${selectedServerStatus.probedAt || ''}`
              "
            >
              <Tag :color="statusTagColor(selectedServerStatus)">
                {{ statusLabel(selectedServerStatus) }}
              </Tag>
            </Tooltip>
            <Button :loading="loadingServerStatus" @click="handleRefreshServerStatus">
              探测状态
            </Button>
            <Button :loading="loadingConfig" @click="fetchConfigSites">刷新站点</Button>
            <Button :loading="loadingSiteMetrics" @click="handleRefreshSiteMetrics">
              刷新指标
            </Button>
            <Button type="primary" :disabled="!selectedServerId" @click="() => openConfigWorkbench()">
              配置工作台
            </Button>
            <Button :disabled="!selectedServerId" @click="openWaf">WAF</Button>
          </Space>
        </template>

        <Alert
          class="mb-3"
          type="info"
          show-icon
          message="目录只读聚合既有 Caddy 配置与 access log 指标；编辑配置 / WAF 请走既有工作台 Preview → 确认 → Apply。"
        />

        <Card size="small" class="server-status-card" :bordered="true">
          <template #title>
            <div class="server-status-title">
              <span>节点状态</span>
              <span v-if="lastProbedAt" class="text-muted">最近探测：{{ lastProbedAt }}</span>
              <span v-else class="text-muted">
                点击「探测状态」检查节点在线与延迟（不会修改运行配置）
              </span>
            </div>
          </template>
          <Alert
            v-if="serverStatusError"
            class="mb-3"
            type="warning"
            show-icon
            :message="serverStatusError"
          />
          <div v-if="servers.length === 0" class="sidebar-empty">
            <Empty description="暂无已登记 Caddy 节点">
              <Button type="primary" @click="() => openConfigWorkbench()">前往配置管理添加</Button>
            </Empty>
          </div>
          <div v-else class="server-status-list">
            <button
              v-for="row in serverStatusRows"
              :key="row.id"
              type="button"
              class="server-status-item"
              :class="{
                active: row.id === selectedServerId,
                online: row.status?.reachable,
                offline: row.status && !row.status.reachable,
              }"
              @click="selectServerFromStatus(row.id)"
            >
              <div class="server-status-main">
                <div class="server-status-name-line">
                  <span class="server-status-name">{{ row.name }}</span>
                  <Tag :color="statusTagColor(row.status)">{{ statusLabel(row.status) }}</Tag>
                </div>
                <div class="server-status-url text-muted">{{ row.url || '—' }}</div>
              </div>
              <div class="server-status-meta">
                <div class="server-status-latency">
                  <span class="meta-label">延迟</span>
                  <strong>{{
                    serverStatusLoaded || row.status
                      ? formatLatency(row.status?.latencyMs)
                      : '未探测'
                  }}</strong>
                </div>
                <div
                  v-if="row.status && !row.status.reachable"
                  class="server-status-error text-error"
                >
                  {{ statusErrorSummary(row.status) || '探测失败' }}
                </div>
                <div v-else-if="row.status?.probedAt" class="server-status-probed text-muted">
                  {{ row.status.probedAt }}
                </div>
              </div>
            </button>
          </div>
        </Card>

        <div class="catalog-toolbar">
          <div class="catalog-summary">
            <Tag>站点 {{ summary.total }}</Tag>
            <Tag color="blue">简单 {{ summary.simple }}</Tag>
            <Tag color="orange">复杂 {{ summary.complex }}</Tag>
            <Tag v-if="serverStatusLoaded" color="success">在线 {{ summary.online }}</Tag>
            <Tag v-if="serverStatusLoaded && summary.offline" color="error">
              离线 {{ summary.offline }}
            </Tag>
            <span class="text-muted">近窗 {{ SITE_METRICS_WINDOW_MINUTES }} 分钟 5xx/4xx</span>
          </div>

          <div class="catalog-filters">
            <Space wrap>
              <Input
                v-model:value="keyword"
                allow-clear
                placeholder="搜索站点名称 / 域名 / 上游"
                style="width: 260px"
              />
              <Select
                v-model:value="kindFilter"
                style="width: 140px"
                :options="[
                  { label: '全部类型', value: 'all' },
                  { label: '简单站点', value: 'simple' },
                  { label: '复杂站点', value: 'complex' },
                ]"
              />
            </Space>
          </div>
        </div>

        <Alert
          v-if="siteMetricsError"
          class="mb-3"
          type="warning"
          show-icon
          :message="`${siteMetricsError}（已降级显示，不影响浏览）`"
        />

        <Spin :spinning="loadingConfig || loadingServers">
          <div v-if="!selectedServerId" class="sidebar-empty">
            <Empty description="请选择 Caddy 节点以查看服务目录" />
          </div>
          <div v-else-if="filteredSites.length === 0" class="sidebar-empty">
            <Empty :description="siteCards.length === 0 ? '当前节点暂无站点' : '无匹配站点'">
              <Space>
                <Button @click="() => openConfigWorkbench()">打开配置工作台</Button>
                <Button @click="openWaf">WAF 入口</Button>
              </Space>
            </Empty>
          </div>
          <Row v-else :gutter="[16, 16]">
            <Col
              v-for="site in filteredSites"
              :key="site.id"
              :xs="24"
              :sm="12"
              :lg="8"
              :xl="6"
            >
              <Card
                size="small"
                class="site-card"
                :class="{ complex: site.kind === 'complex' }"
                hoverable
                @click="openSiteInWorkbench(site)"
              >
                <div class="site-card-header">
                  <div class="site-card-title">
                    <span class="site-name">{{ site.name || '未命名站点' }}</span>
                    <Tag :color="site.kind === 'simple' ? 'blue' : 'orange'">
                      {{ site.kind === 'simple' ? '简单' : '复杂' }}
                    </Tag>
                  </div>
                  <Tag :color="site.enabled ? 'green' : 'default'">
                    {{ site.enabled ? '启用' : '停用' }}
                  </Tag>
                </div>

                <div class="site-domain text-muted">
                  {{ site.domains[0] || site.primaryHost || '未配置域名' }}
                </div>
                <div v-if="site.domains.length > 1" class="site-domain-extra text-muted">
                  +{{ site.domains.length - 1 }} 个域名
                </div>

                <div class="site-meta-row">
                  <Tag>{{ catalogModeLabel(site.mode) }}</Tag>
                  <span v-if="site.upstream" class="text-muted site-upstream">
                    {{ site.upstream }}
                  </span>
                </div>

                <div v-if="site.lbPolicy || site.healthPath" class="site-extra text-muted">
                  <span v-if="site.lbPolicy">lb: {{ site.lbPolicy }}</span>
                  <span v-if="site.healthPath">health: {{ site.healthPath }}</span>
                </div>

                <div class="site-metrics">
                  <span class="site-metric">
                    <span class="meta-label">近窗 5xx</span>
                    <Tag
                      class="site-metric-tag"
                      :color="
                        metricCountColor(
                          siteMetricsForHost(siteMetricsMap, site.domains)?.count5xx,
                          '5xx',
                        )
                      "
                    >
                      {{ metricDisplay(site, '5xx') }}
                    </Tag>
                  </span>
                  <span class="site-metric">
                    <span class="meta-label">近窗 4xx</span>
                    <Tag
                      class="site-metric-tag"
                      :color="
                        metricCountColor(
                          siteMetricsForHost(siteMetricsMap, site.domains)?.count4xx,
                          '4xx',
                        )
                      "
                    >
                      {{ metricDisplay(site, '4xx') }}
                    </Tag>
                  </span>
                </div>

                <div v-if="site.reasons?.length" class="site-reasons text-muted">
                  {{ site.reasons[0] }}
                  <span v-if="site.reasons.length > 1"> 等 {{ site.reasons.length }} 条原因</span>
                </div>

                <div class="site-actions" @click.stop>
                  <Button
                    v-if="sitePrimaryHost(site.domains)"
                    size="small"
                    type="link"
                    @click="(e: Event) => openAccessLogs(site.domains, e)"
                  >
                    查看日志
                  </Button>
                  <Button size="small" type="link" @click="openSiteInWorkbench(site)">
                    配置
                  </Button>
                  <Button size="small" type="link" @click="openWaf">WAF</Button>
                </div>
              </Card>
            </Col>
          </Row>
        </Spin>

        <div class="catalog-secondary mt-4">
          <Space wrap>
            <span class="text-muted">辅助入口（仍走既有 Apply_Path）：</span>
            <Button size="small" @click="() => openConfigWorkbench('blocks')">站点向导 / 编辑</Button>
            <Button size="small" @click="() => openConfigWorkbench()">Docker 发现（工作台内）</Button>
            <Button size="small" @click="openWaf">防火墙 WAF</Button>
          </Space>
        </div>
      </Card>
    </div>
  </Page>
</template>

<style scoped>
.service-catalog-page {
  padding: 16px;
}

.catalog-shell {
  min-height: calc(100vh - 140px);
}

.catalog-title-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.catalog-subtitle {
  font-size: 12px;
  font-weight: 400;
  color: #8c8c8c;
}

.server-select {
  min-width: 220px;
}

.server-status-card {
  margin-bottom: 24px;
  border-color: #eef0f4;
  border-radius: 8px;
}

.server-status-title {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: baseline;
}

.server-status-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
}

.server-status-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 96px;
  padding: 12px 14px;
  text-align: left;
  cursor: pointer;
  background: #fbfcfd;
  border: 1px solid #eef0f4;
  border-radius: 8px;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    background 0.16s ease;
}

.server-status-item:hover {
  border-color: #c7d7fe;
}

.server-status-item.active {
  background: #f0f7ff;
  border-color: #1677ff;
  box-shadow: 0 0 0 2px rgb(22 119 255 / 8%);
}

.server-status-item.online {
  border-left: 3px solid #52c41a;
}

.server-status-item.offline {
  border-left: 3px solid #ff4d4f;
}

.server-status-name-line {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
}

.server-status-name {
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.server-status-url {
  margin-top: 4px;
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.server-status-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.server-status-latency {
  display: flex;
  gap: 8px;
  align-items: baseline;
}

.server-status-error {
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.catalog-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  padding: 12px 14px;
  background: #fafbfc;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
}

.catalog-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.catalog-filters {
  flex-shrink: 0;
}

.site-card {
  height: 100%;
  border-radius: 10px;
  transition: border-color 0.16s ease;
}

.site-card.complex {
  border-color: #ffd591;
}

.site-card-header {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 6px;
}

.site-card-title {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  min-width: 0;
}

.site-name {
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-domain {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-domain-extra {
  margin-top: 2px;
  font-size: 12px;
}

.site-meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-top: 8px;
}

.site-upstream {
  overflow: hidden;
  max-width: 100%;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-extra {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 6px;
  font-size: 12px;
}

.site-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-top: 12px;
}

.site-metric {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.meta-label {
  font-size: 12px;
  color: #8c8c8c;
}

.site-metric-tag {
  margin-inline-end: 0;
}

.site-reasons {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.4;
}

.site-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0;
  margin-top: 8px;
  margin-left: -8px;
}

.sidebar-empty {
  padding: 24px 0;
}

.text-muted {
  color: #8c8c8c;
}

.text-error {
  color: #ff4d4f;
}

.catalog-secondary {
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}
</style>
