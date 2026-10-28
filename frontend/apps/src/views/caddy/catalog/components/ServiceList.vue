<script lang="ts" setup>
/**
 * 站点卡片网格（对齐雷池防护应用解剖）：
 * 站点名 + 更多菜单 → 防护模式徽章 → 域名 → 今日请求/拦截 → 能力 chips（IP 限制/证书天数）。
 * 卡片主体点击进详情整页；低频动作（日志/原文/WAF）收进「更多」下拉。
 */
import {
  Alert,
  Button,
  Card,
  Col,
  Dropdown,
  Empty,
  Input,
  Row,
  Space,
  Spin,
  Tag,
  Tooltip,
} from 'antdv-next';

import type { CaddyCertificateApi } from '#/api/caddy/certificate';

import {
  catalogModeLabel,
  siteCertDaysLeft,
  siteMetricsForDomains,
  siteProtectBadge,
  sitePrimaryHost,
  type CatalogSiteCard,
  type SiteMetricsMap,
} from '../service-catalog-utils';
import { describeSiteCapabilities } from '../../config/site-capabilities';

const props = defineProps<{
  loading: boolean;
  keyword: string;
  siteCardsCount: number;
  filteredSites: CatalogSiteCard[];
  siteMetricsError: string;
  siteMetricsMap: SiteMetricsMap;
  certList: CaddyCertificateApi.CertificateItem[];
  wafStatus: { enabled?: boolean; mode?: string } | null;
  todayDisplay: (site: CatalogSiteCard, kind: 'total' | 'blocked') => string;
}>();

const emit = defineEmits<{
  'update:keyword': [value: string];
  refresh: [];
  addSite: [];
  openSite: [site: CatalogSiteCard];
  openAccessLogs: [domains: string[] | undefined, event?: Event];
  openWaf: [];
  openWorkbench: [];
}>();

function onKeywordChange(value: string) {
  emit('update:keyword', value);
}

/** 卡片「更多」菜单项：查看日志需有可过滤域名，其余常开 */
function moreItems(site: CatalogSiteCard) {
  return [
    { key: 'logs', label: '查看日志', disabled: !sitePrimaryHost(site.domains) },
    { key: 'raw', label: '原文编辑（工作台）' },
    { key: 'waf', label: 'WAF 策略' },
  ];
}

function handleMoreClick(info: { key: number | string }, site: CatalogSiteCard) {
  if (info.key === 'logs') {
    emit('openAccessLogs', site.domains);
  } else if (info.key === 'raw') {
    emit('openWorkbench');
  } else if (info.key === 'waf') {
    emit('openWaf');
  }
}

/** 防护模式徽章：站点挂载状态 × 全局 WAF 引擎状态 */
function protectBadge(site: CatalogSiteCard) {
  return siteProtectBadge(site, props.wafStatus);
}

/** 证书天数 chip：按域名匹配证书库存取最小剩余天数；未命中不展示 */
function certDaysLeft(site: CatalogSiteCard) {
  return siteCertDaysLeft(props.certList, site.domains);
}

function certChip(days: number) {
  if (days < 0) return { label: '证书已过期', color: 'error' };
  if (days <= 14) return { label: `证书 ${days} 天`, color: 'warning' };
  return { label: `证书 ${days} 天`, color: 'success' };
}

/**
 * 能力 chips：只留 IP 限制；WAF 挂载已由防护徽章表达，仅 unknown 时提示注意；
 * 安全头/错误页/压缩/日志在详情页防护与高级 Tab 维护。
 */
function capabilityBadges(site: CatalogSiteCard) {
  if (!site.capabilities) return [];
  return describeSiteCapabilities(site.capabilities).filter(
    (item) => item.key === 'geo' || (item.key === 'waf' && item.status === 'unknown'),
  );
}

/** 今日拦截原始值：> 0 时数值标红 */
function blockedCount(site: CatalogSiteCard): number {
  return siteMetricsForDomains(props.siteMetricsMap, site.domains)?.blocked ?? 0;
}

/** 今日 5xx 原始值：> 0 说明上游异常，是比拦截更该被看见的信号 */
function serverErrorCount(site: CatalogSiteCard): number {
  return siteMetricsForDomains(props.siteMetricsMap, site.domains)?.count5xx ?? 0;
}

/** 卡片左上角字母头像：取站点名首个可见字符 */
function avatarText(site: CatalogSiteCard): string {
  const source = (site.name || sitePrimaryHost(site.domains) || '?').trim();
  return (source[0] ?? '?').toUpperCase();
}

/** 卡片顶部色条：与防护徽章同语义，让整列卡片的防护状态可以一眼扫过 */
function accentColor(site: CatalogSiteCard): string {
  const badge = protectBadge(site);
  if (badge.label === '阻断') return 'hsl(var(--success))';
  if (badge.label === '检测' || badge.label === '未生效') return 'hsl(var(--warning))';
  if (badge.label === '未防护' || badge.label === 'WAF?') return 'hsl(var(--destructive))';
  return 'hsl(var(--border))';
}

/** 拦截占比：正文小字，请求为 0 时不显示，避免 0/0 噪音 */
function blockRate(site: CatalogSiteCard): string {
  const metrics = siteMetricsForDomains(props.siteMetricsMap, site.domains);
  const total = metrics?.total ?? 0;
  const blocked = metrics?.blocked ?? 0;
  if (total <= 0 || blocked <= 0) return '';
  const rate = (blocked / total) * 100;
  return rate < 0.1 ? '<0.1%' : `${rate.toFixed(rate < 10 ? 1 : 0)}%`;
}
</script>

<template>
  <div class="service-list">
    <div class="catalog-toolbar">
      <Input
        :value="props.keyword"
        allow-clear
        placeholder="搜索站点名称 / 域名 / 上游"
        class="toolbar-search"
        @update:value="onKeywordChange"
      />
      <span class="text-muted toolbar-summary">
        共 {{ props.siteCardsCount }} 个站点 · 今日（24 小时）统计
      </span>
      <div class="toolbar-actions">
        <Button :loading="props.loading" @click="emit('refresh')">刷新</Button>
        <Button type="primary" @click="emit('addSite')">添加站点</Button>
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
      <div v-if="props.filteredSites.length === 0" class="sidebar-empty">
        <Empty :description="props.siteCardsCount === 0 ? '暂无站点' : '无匹配站点'">
          <Space>
            <Button type="primary" @click="emit('addSite')">添加站点</Button>
            <Button @click="emit('openWorkbench')">打开配置工作台</Button>
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
            <span class="site-card-accent" :style="{ background: accentColor(site) }"></span>

            <div class="site-card-header">
              <span class="site-avatar">{{ avatarText(site) }}</span>
              <div class="site-title">
                <span class="site-name" :title="site.name">
                  {{ site.name || '未命名站点' }}
                </span>
                <span
                  v-if="sitePrimaryHost(site.domains) !== site.name"
                  class="site-domain"
                  :title="site.domains.join('、')"
                >
                  {{ site.domains[0] || site.primaryHost || '未配置域名' }}
                  <span v-if="site.domains.length > 1">
                    +{{ site.domains.length - 1 }}
                  </span>
                </span>
              </div>
              <div class="site-more" @click.stop>
                <Dropdown
                  :trigger="['click']"
                  placement="bottomRight"
                  :menu="{
                    items: moreItems(site),
                    onClick: (info: { key: number | string }) =>
                      handleMoreClick(info, site),
                  }"
                >
                  <Button size="small" type="text">更多</Button>
                </Dropdown>
              </div>
            </div>

            <div class="site-badges">
              <Tooltip :title="protectBadge(site).title">
                <Tag :color="protectBadge(site).color">
                  {{ protectBadge(site).label }}
                </Tag>
              </Tooltip>
              <Tag class="cap-tag">{{ catalogModeLabel(site.mode) }}</Tag>
              <Tooltip v-if="serverErrorCount(site) > 0" title="近 24 小时站点返回 5xx 的次数，通常意味着上游不可用">
                <Tag class="cap-tag" color="error">5xx {{ serverErrorCount(site) }}</Tag>
              </Tooltip>
            </div>

            <div class="site-stats">
              <div class="stat">
                <span class="stat-value">{{ props.todayDisplay(site, 'total') }}</span>
                <span class="stat-label">今日请求</span>
              </div>
              <div class="stat-divider"></div>
              <div class="stat">
                <span
                  class="stat-value"
                  :class="{ danger: blockedCount(site) > 0 }"
                >
                  {{ props.todayDisplay(site, 'blocked') }}
                </span>
                <span class="stat-label">
                  今日拦截
                  <span v-if="blockRate(site)" class="stat-rate">{{ blockRate(site) }}</span>
                </span>
              </div>
            </div>

            <div
              v-if="capabilityBadges(site).length || certDaysLeft(site) !== null"
              class="site-caps"
            >
              <Tooltip
                v-for="badge in capabilityBadges(site)"
                :key="badge.key"
                :title="
                  badge.status === 'unknown'
                    ? '检测到无法识别的配置，开关已禁用'
                    : badge.label
                "
              >
                <Tag
                  class="cap-tag"
                  :color="badge.status === 'unknown' ? 'default' : 'geekblue'"
                >
                  {{ badge.label }}{{ badge.status === 'unknown' ? '?' : '' }}
                </Tag>
              </Tooltip>
              <Tooltip
                v-if="certDaysLeft(site) !== null"
                title="按域名匹配的证书剩余天数（多证书取最小值）"
              >
                <Tag class="cap-tag" :color="certChip(certDaysLeft(site)!).color">
                  {{ certChip(certDaysLeft(site)!).label }}
                </Tag>
              </Tooltip>
            </div>

            <div v-if="site.upstream" class="site-upstream text-muted" :title="site.upstream">
              {{ site.upstream }}
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
  margin-bottom: 20px;
  padding: 12px 14px;
  background: hsl(var(--muted) / 40%);
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
}

.toolbar-search {
  width: 260px;
}

.toolbar-summary {
  flex: 1;
  min-width: 0;
  font-size: 12px;
}

.toolbar-actions {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
}

.site-card {
  position: relative;
  height: 100%;
  padding-top: 4px;
  overflow: hidden;
  border-radius: 12px;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.site-card:hover {
  border-color: hsl(var(--primary) / 35%);
  box-shadow: 0 6px 20px hsl(var(--foreground) / 8%);
  transform: translateY(-1px);
}

.site-card.complex {
  border-color: hsl(var(--warning) / 50%);
}

/* 顶部分状态色条：阻断=绿、检测/未生效=橙、未防护=红、未知=灰 */
.site-card-accent {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
}

.site-card-header {
  display: flex;
  gap: 10px;
  align-items: center;
}

.site-avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  font-size: 15px;
  font-weight: 600;
  color: hsl(var(--primary));
  background: hsl(var(--primary) / 10%);
  border-radius: 9px;
}

.site-title {
  display: flex;
  overflow: hidden;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  line-height: 1.35;
}

.site-name {
  overflow: hidden;
  font-size: 14px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-more {
  flex-shrink: 0;
}

.site-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  margin-top: 10px;
}

.site-domain {
  overflow: hidden;
  font-size: 12px;
  /* muted-foreground 在浅色主题下过淡（截图反馈），改用主前景色 72% 透明度保证可读 */
  color: hsl(var(--foreground) / 72%);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-stats {
  display: flex;
  gap: 16px;
  align-items: center;
  margin-top: 12px;
  padding: 10px 12px;
  background: hsl(var(--muted) / 45%);
  border-radius: 10px;
}

.stat {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.stat-divider {
  align-self: stretch;
  width: 1px;
  background: hsl(var(--border));
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1.25;
}

.stat-value.danger {
  color: hsl(var(--destructive));
}

.stat-label {
  display: flex;
  gap: 4px;
  align-items: baseline;
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}

.stat-rate {
  font-variant-numeric: tabular-nums;
  opacity: 0.8;
}

.site-caps {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 8px;
}

.cap-tag {
  margin-inline-end: 0;
  font-size: 11px;
}

.site-upstream {
  overflow: hidden;
  margin-top: 8px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-empty {
  padding: 24px 0;
}

.text-muted {
  color: hsl(var(--muted-foreground));
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
