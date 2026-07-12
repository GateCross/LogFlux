<script lang="ts" setup>
/** 目录站点列表（只读；配置/WAF/日志仅导航） */
import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Input,
  Row,
  Select,
  Space,
  Spin,
  Tag,
} from 'antdv-next';

import {
  SITE_METRICS_WINDOW_MINUTES,
  catalogModeLabel,
  metricCountColor,
  siteMetricsForHost,
  sitePrimaryHost,
  type CatalogSiteCard,
  type SiteMetricsMap,
} from '../service-catalog-utils';

const props = defineProps<{
  selectedServerId?: number;
  loading: boolean;
  keyword: string;
  kindFilter: 'all' | 'simple' | 'complex';
  summary: {
    total: number;
    simple: number;
    complex: number;
    online: number;
    offline: number;
  };
  serverStatusLoaded: boolean;
  siteMetricsError: string;
  siteCardsCount: number;
  filteredSites: CatalogSiteCard[];
  siteMetricsMap: SiteMetricsMap;
  metricDisplay: (site: CatalogSiteCard, kind: '4xx' | '5xx') => string;
}>();

const emit = defineEmits<{
  'update:keyword': [value: string];
  'update:kindFilter': [value: 'all' | 'simple' | 'complex'];
  openSite: [site: CatalogSiteCard];
  openAccessLogs: [domains: string[] | undefined, event?: Event];
  openWaf: [];
  openWorkbench: [];
}>();

function onKeywordChange(value: string) {
  emit('update:keyword', value);
}

function onKindFilterChange(value: unknown) {
  if (value === 'all' || value === 'simple' || value === 'complex') {
    emit('update:kindFilter', value);
  }
}
</script>

<template>
  <div class="service-list">
    <div class="catalog-toolbar">
      <div class="catalog-summary">
        <Tag>站点 {{ props.summary.total }}</Tag>
        <Tag color="blue">简单 {{ props.summary.simple }}</Tag>
        <Tag color="orange">复杂 {{ props.summary.complex }}</Tag>
        <Tag v-if="props.serverStatusLoaded" color="success">
          在线 {{ props.summary.online }}
        </Tag>
        <Tag v-if="props.serverStatusLoaded && props.summary.offline" color="error">
          离线 {{ props.summary.offline }}
        </Tag>
        <span class="text-muted">近窗 {{ SITE_METRICS_WINDOW_MINUTES }} 分钟 5xx/4xx</span>
      </div>

      <div class="catalog-filters">
        <Space wrap>
          <Input
            :value="props.keyword"
            allow-clear
            placeholder="搜索站点名称 / 域名 / 上游"
            style="width: 260px"
            @update:value="onKeywordChange"
          />
          <Select
            :value="props.kindFilter"
            style="width: 140px"
            :options="[
              { label: '全部类型', value: 'all' },
              { label: '简单站点', value: 'simple' },
              { label: '复杂站点', value: 'complex' },
            ]"
            @change="onKindFilterChange"
          />
        </Space>
      </div>
    </div>

    <Alert
      v-if="props.siteMetricsError"
      class="mb-3"
      type="warning"
      show-icon
      :message="`${props.siteMetricsError}（已降级显示，不影响浏览）`"
    />

    <Spin :spinning="props.loading">
      <div v-if="!props.selectedServerId" class="sidebar-empty">
        <Empty description="请选择 Caddy 节点以查看服务目录" />
      </div>
      <div v-else-if="props.filteredSites.length === 0" class="sidebar-empty">
        <Empty
          :description="props.siteCardsCount === 0 ? '当前节点暂无站点' : '无匹配站点'"
        >
          <Space>
            <Button @click="emit('openWorkbench')">打开配置工作台</Button>
            <Button @click="emit('openWaf')">WAF 入口</Button>
          </Space>
        </Empty>
      </div>
      <Row v-else :gutter="[16, 16]">
        <Col
          v-for="site in props.filteredSites"
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
            @click="emit('openSite', site)"
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
                      siteMetricsForHost(props.siteMetricsMap, site.domains)?.count5xx,
                      '5xx',
                    )
                  "
                >
                  {{ props.metricDisplay(site, '5xx') }}
                </Tag>
              </span>
              <span class="site-metric">
                <span class="meta-label">近窗 4xx</span>
                <Tag
                  class="site-metric-tag"
                  :color="
                    metricCountColor(
                      siteMetricsForHost(props.siteMetricsMap, site.domains)?.count4xx,
                      '4xx',
                    )
                  "
                >
                  {{ props.metricDisplay(site, '4xx') }}
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
                @click="(e: Event) => emit('openAccessLogs', site.domains, e)"
              >
                查看日志
              </Button>
              <Button size="small" type="link" @click="emit('openSite', site)">
                配置
              </Button>
              <Button size="small" type="link" @click="emit('openWaf')">WAF</Button>
            </div>
          </Card>
        </Col>
      </Row>
    </Spin>
  </div>
</template>

<style scoped>
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

.mb-3 {
  margin-bottom: 12px;
}
</style>
