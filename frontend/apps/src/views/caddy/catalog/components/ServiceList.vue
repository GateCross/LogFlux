<script lang="ts" setup>
/**
 * 站点卡片网格（对齐雷池防护应用解剖）：
 * 左栏是安全态势——站点图标、防护模式徽章、今日请求/拦截；
 * 右栏是站点资产——名称、域名、端口、上游、能力胶囊。
 * 卡片主体点击进详情整页；低频动作（日志/原文/WAF）收进「更多」下拉。
 */
import type { CaddyCertificateApi } from '#/api/caddy/certificate';

import { Icon } from '@iconify/vue/offline';
import { Alert, Button, Card, Col, Dropdown, Empty, Input, Row, Space, Spin, Tooltip } from 'antdv-next';

import { describeSiteCapabilities } from '../../config/site-capabilities';
import SiteAvatar from '../../shared/components/SiteAvatar.vue';
import {
  catalogModeLabel,
  type CatalogSiteCard,
  normalizeSiteHost,
  siteCertDaysLeft,
  siteMetricsForDomains,
  type SiteMetricsMap,
  sitePrimaryHost,
  siteProtectBadge,
} from '../service-catalog-utils';

const props = defineProps<{
  certList: CaddyCertificateApi.CertificateItem[];
  filteredSites: CatalogSiteCard[];
  keyword: string;
  loading: boolean;
  siteCardsCount: number;
  siteMetricsError: string;
  siteMetricsMap: SiteMetricsMap;
  todayDisplay: (site: CatalogSiteCard, kind: 'blocked' | 'total') => string;
}>();

const emit = defineEmits<{
  addSite: [];
  openAccessLogs: [domains: string[] | undefined, event?: Event];
  openSite: [site: CatalogSiteCard];
  openWaf: [];
  openWorkbench: [];
  refresh: [];
  'update:keyword': [value: string];
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
  switch (info.key) {
    case 'logs': {
      emit('openAccessLogs', site.domains);
      break;
    }
    case 'raw': {
      emit('openWorkbench');
      break;
    }
    case 'waf': {
      emit('openWaf');
      break;
    }
  }
}

/** 防护模式徽章：全部由站点自己的能力快照推导，站点级、彼此独立 */
function protectBadge(site: CatalogSiteCard) {
  return siteProtectBadge(site);
}

/** 徽章配色：绿=在拦、琥珀=只看不拦、粉红=维护模式、红=没防护、灰=状态不明 */
function modeStyle(site: CatalogSiteCard): Record<string, string> {
  const { label } = protectBadge(site);
  if (label === '维护模式') {
    return {
      color: 'hsl(var(--destructive))',
      background: 'hsl(var(--destructive) / 12%)',
      borderColor: 'hsl(var(--destructive) / 45%)',
    };
  }
  if (label === '防护模式') {
    return {
      color: 'hsl(var(--success))',
      background: 'hsl(var(--success) / 12%)',
      borderColor: 'hsl(var(--success) / 40%)',
    };
  }
  if (label === '观察模式' || label === '未生效') {
    return {
      color: 'hsl(var(--warning))',
      background: 'hsl(var(--warning) / 14%)',
      borderColor: 'hsl(var(--warning) / 42%)',
    };
  }
  if (label === '未防护') {
    return {
      color: 'hsl(var(--destructive))',
      background: 'hsl(var(--destructive) / 10%)',
      borderColor: 'hsl(var(--destructive) / 35%)',
    };
  }
  return {
    color: 'hsl(var(--muted-foreground))',
    background: 'hsl(var(--muted) / 60%)',
    borderColor: 'hsl(var(--border))',
  };
}

/** 卡片顶部色条：与防护徽章同语义，让整列卡片的防护状态可以一眼扫过 */
function accentColor(site: CatalogSiteCard): string {
  const { label } = protectBadge(site);
  if (label === '维护模式') return 'hsl(var(--destructive))';
  if (label === '防护模式') return 'hsl(var(--success))';
  if (label === '观察模式' || label === '未生效') return 'hsl(var(--warning))';
  if (label === '未防护') return 'hsl(var(--destructive))';
  return 'hsl(var(--border))';
}

function certDaysLeft(site: CatalogSiteCard) {
  return siteCertDaysLeft(props.certList, site.domains);
}

function certChip(days: number) {
  if (days < 0) return { label: '证书已过期', tone: 'danger', title: '证书已过期，请尽快续期' };
  if (days <= 14) return { label: `证书 ${days} 天`, tone: 'warn', title: '证书即将到期，请及时续期' };
  return { label: `证书 ${days} 天`, tone: 'ok', title: '按域名匹配的证书剩余天数（多证书取最小值）' };
}

/** 证书 tone 到胶囊状态的映射，避免在对象字面量里写嵌套三元 */
function certChipState(tone: string): CapabilityChip['state'] {
  if (tone === 'danger') return 'danger';
  if (tone === 'warn') return 'warn';
  return 'on';
}

/** 今日拦截原始值：> 0 时数值标红 */
function blockedCount(site: CatalogSiteCard): number {
  return siteMetricsForDomains(props.siteMetricsMap, site.domains)?.blocked ?? 0;
}

/** 今日 5xx 原始值：> 0 说明上游异常，是比拦截更该被看见的信号 */
function serverErrorCount(site: CatalogSiteCard): number {
  return siteMetricsForDomains(props.siteMetricsMap, site.domains)?.count5xx ?? 0;
}

/** 卡片左上角头像：域名自带图标，取不到时回落站点名首字符 */
function avatarHost(site: CatalogSiteCard): string {
  return sitePrimaryHost(site.domains);
}

/** 主域名展示：多个域名时补 +N */
function domainText(site: CatalogSiteCard): string {
  const host = sitePrimaryHost(site.domains);
  if (!host) return '未配置域名';
  const extra = site.domains.length > 1 ? ` +${site.domains.length - 1}` : '';
  return `${host}${extra}`;
}

/**
 * 协议：地址里显式写了就照抄，否则按 TLS 能力推断。
 * Caddy 未写 tls off 时默认走自动 HTTPS，internal 也只是换成内部 CA，同样算 HTTPS。
 */
function protocolOf(site: CatalogSiteCard): 'HTTP' | 'HTTPS' {
  const raw = (site.domains[0] ?? '').toLowerCase();
  if (raw.startsWith('http://')) return 'HTTP';
  if (raw.startsWith('https://')) return 'HTTPS';
  return site.capabilities?.tls.mode === 'off' ? 'HTTP' : 'HTTPS';
}

/** 端口：地址里带显式端口就用它，否则按协议取默认端口 */
function portOf(site: CatalogSiteCard): string {
  const raw = (site.domains[0] ?? '').trim();
  const withoutScheme = raw.replace(/^[a-zA-Z][\w+.-]*:\/\//, '');
  const hostPart = withoutScheme.split('/')[0] ?? '';
  // IPv6 字面量里的冒号不是端口分隔符
  const match = hostPart.includes(']')
    ? hostPart.slice(hostPart.indexOf(']')).match(/^]:(\d+)$/)
    : hostPart.match(/:(\d+)$/);
  if (match?.[1]) return match[1];
  return protocolOf(site) === 'HTTPS' ? '443' : '80';
}

function portText(site: CatalogSiteCard): string {
  return `${portOf(site)}/${protocolOf(site)}`;
}

interface CapabilityChip {
  key: string;
  label: string;
  state: 'danger' | 'on' | 'unknown' | 'warn';
  title: string;
}

/**
 * 能力胶囊：只列已开启与无法识别的项。
 * 未开启的能力是「没装的护盾」，铺成一片灰字只会稀释真正亮着的那几个。
 */
function capabilityChips(site: CatalogSiteCard): CapabilityChip[] {
  const chips: CapabilityChip[] = [];
  if (site.capabilities) {
    for (const item of describeSiteCapabilities(site.capabilities)) {
      // WAF 与维护模式已由左栏模式徽章表达，这里不重复
      if (item.key === 'waf' || item.key === 'maintenance') continue;
      if (item.status === 'on') {
        chips.push({ key: item.key, label: item.label, state: 'on', title: `${item.label} 已启用` });
      } else if (item.status === 'unknown') {
        chips.push({
          key: item.key,
          label: `${item.label}?`,
          state: 'unknown',
          title: '检测到无法识别的配置，请在详情页确认',
        });
      }
    }
  }
  const days = certDaysLeft(site);
  if (days !== null) {
    const chip = certChip(days);
    chips.push({
      key: 'cert',
      label: chip.label,
      state: certChipState(chip.tone),
      title: chip.title,
    });
  }
  if (serverErrorCount(site) > 0) {
    chips.push({
      key: '5xx',
      label: `5xx ${serverErrorCount(site)}`,
      state: 'danger',
      title: '近 24 小时站点返回 5xx 的次数，通常意味着上游不可用',
    });
  }
  return chips;
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

/** 多域名 +N 的悬浮提示：补全被折叠的其余域名 */
function domainsTitle(site: CatalogSiteCard): string {
  const list = site.domains
    .map((domain) => normalizeSiteHost(domain) || domain)
    .filter(Boolean);
  return list.length > 0 ? list.join('、') : '未配置域名';
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
          :sm="24"
          :lg="12"
          :xl="8"
          :xxl="8"
        >
          <Card
            size="small"
            class="site-card"
            :class="{ complex: site.kind === 'complex' }"
            hoverable
            @click="emit('openSite', site)"
          >
            <span class="site-card-accent" :style="{ background: accentColor(site) }"></span>

            <!-- 左栏：安全态势 -->
            <div class="site-rail">
              <SiteAvatar :host="avatarHost(site)" :name="site.name" :size="40" />

              <Tooltip :title="protectBadge(site).title">
                <span class="mode-badge" :style="modeStyle(site)">
                  <span class="mode-dot"></span>
                  {{ protectBadge(site).label }}
                </span>
              </Tooltip>

              <div class="rail-stats">
                <div class="rail-stat">
                  <span class="rail-value">{{ props.todayDisplay(site, 'total') }}</span>
                  <span class="rail-label">今日请求</span>
                </div>
                <span class="rail-divider"></span>
                <div class="rail-stat">
                  <span class="rail-value" :class="{ danger: blockedCount(site) > 0 }">
                    {{ props.todayDisplay(site, 'blocked') }}
                  </span>
                  <span class="rail-label">
                    今日拦截
                    <span v-if="blockRate(site)" class="rail-rate">{{ blockRate(site) }}</span>
                  </span>
                </div>
              </div>
            </div>

            <!-- 右栏：站点资产 -->
            <div class="site-main">
              <div class="main-head">
                <div class="main-title">
                  <span class="site-name" :title="site.name">
                    {{ site.name || '未命名站点' }}
                  </span>
                  <span class="site-kind">{{ catalogModeLabel(site.mode) }}</span>
                </div>
                <div class="main-actions" @click.stop>
                  <Button type="link" size="small" class="detail-link" @click="emit('openSite', site)">
                    详情
                  </Button>
                  <Dropdown
                    :trigger="['click']"
                    placement="bottomRight"
                    :menu="{
                      items: moreItems(site),
                      onClick: (info: { key: number | string }) => handleMoreClick(info, site),
                    }"
                  >
                    <Button size="small" type="text" class="more-btn">
                      <Icon icon="mdi:dots-horizontal" />
                    </Button>
                  </Dropdown>
                </div>
              </div>

              <div class="meta-list">
                <div class="meta-row">
                  <Icon class="meta-icon" icon="mdi:earth" />
                  <span class="meta-label">域名</span>
                  <span class="meta-value" :title="domainsTitle(site)">{{ domainText(site) }}</span>
                </div>
                <div class="meta-row">
                  <Icon class="meta-icon" icon="mdi:lan-connect" />
                  <span class="meta-label">端口</span>
                  <span class="meta-value" :class="{ secure: protocolOf(site) === 'HTTPS' }">
                    {{ portText(site) }}
                  </span>
                </div>
                <div v-if="site.upstream" class="meta-row">
                  <Icon class="meta-icon" icon="mdi:server-network" />
                  <span class="meta-label">上游</span>
                  <span class="meta-value mono" :title="site.upstream">{{ site.upstream }}</span>
                </div>
              </div>

              <div v-if="capabilityChips(site).length > 0" class="chip-row">
                <Tooltip v-for="chip in capabilityChips(site)" :key="chip.key" :title="chip.title">
                  <span class="chip" :class="chip.state">{{ chip.label }}</span>
                </Tooltip>
              </div>
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
  overflow: hidden;
  border-radius: 12px;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.site-card :deep(.ant-card-body) {
  display: flex;
  align-items: stretch;
  height: 100%;
  padding: 0;
}

.site-card:hover {
  border-color: hsl(var(--primary) / 35%);
  box-shadow: 0 6px 20px hsl(var(--foreground) / 8%);
  transform: translateY(-1px);
}

.site-card.complex {
  border-color: hsl(var(--warning) / 50%);
}

/* 顶部分状态色条：防护=绿、观察/未生效=橙、未防护=红、未知=灰 */
.site-card-accent {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  z-index: 1;
  height: 3px;
}

/* 左栏：图标 + 模式徽章 + 今日数值，用虚线分隔出一条竖向「安全态势」区 */
.site-rail {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  gap: 9px;
  align-items: center;
  justify-content: center;
  width: 136px;
  padding: 18px 10px 14px;
  background: hsl(var(--muted) / 28%);
  border-right: 1px dashed hsl(var(--border));
}

.mode-badge {
  display: inline-flex;
  gap: 5px;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 3px 6px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.4;
  letter-spacing: 0.02em;
  white-space: nowrap;
  cursor: help;
  border: 1px solid transparent;
  border-radius: 6px;
}

/* 圆点跟随徽章文字色，让「在拦 / 只看 / 没防护」在缩略尺寸下也能区分 */
.mode-dot {
  flex-shrink: 0;
  width: 5px;
  height: 5px;
  background: currentcolor;
  border-radius: 50%;
}

.rail-stats {
  display: flex;
  gap: 4px;
  align-items: center;
  justify-content: center;
  width: 100%;
  margin-top: 4px;
}

.rail-stat {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 1px;
  align-items: center;
  min-width: 0;
}

.rail-divider {
  width: 1px;
  height: 24px;
  background: hsl(var(--border));
}

.rail-value {
  font-size: 16px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.rail-value.danger {
  color: hsl(var(--destructive));
}

.rail-label {
  font-size: 11px;
  color: hsl(var(--muted-foreground));
  white-space: nowrap;
}

.rail-rate {
  font-variant-numeric: tabular-nums;
  opacity: 0.8;
}

/* 右栏：名称 + 资产信息 + 能力胶囊 */
.site-main {
  display: flex;
  overflow: hidden;
  flex: 1;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 16px 16px 14px;
}

.main-head {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  justify-content: space-between;
}

.main-title {
  display: flex;
  gap: 8px;
  align-items: baseline;
  min-width: 0;
}

.site-name {
  overflow: hidden;
  font-size: 15px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-kind {
  flex-shrink: 0;
  font-size: 11px;
  color: hsl(var(--muted-foreground));
}

.main-actions {
  display: flex;
  flex-shrink: 0;
  gap: 6px;
  align-items: center;
}

.detail-link {
  height: auto;
  padding: 0;
  font-size: 12px;
}

.more-btn {
  width: 20px;
  height: 20px;
  padding: 0;
  color: hsl(var(--muted-foreground));
}

.meta-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
}

.meta-row {
  display: flex;
  gap: 6px;
  align-items: center;
  min-width: 0;
  font-size: 12px;
}

.meta-icon {
  flex-shrink: 0;
  font-size: 13px;
  color: hsl(var(--muted-foreground));
}

.meta-label {
  flex-shrink: 0;
  width: 30px;
  color: hsl(var(--muted-foreground));
}

.meta-value {
  overflow: hidden;
  color: hsl(var(--foreground) / 78%);
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* HTTPS/端口用语义色点出来：安全通道一眼可辨 */
.meta-value.secure {
  font-weight: 500;
  color: hsl(var(--success));
}

.meta-value.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: auto;
  padding-top: 2px;
}

.chip {
  padding: 1px 7px;
  font-size: 11px;
  line-height: 1.6;
  cursor: help;
  background: hsl(var(--muted) / 55%);
  border: 1px solid transparent;
  border-radius: 4px;
}

.chip.on {
  color: hsl(var(--success));
  background: hsl(var(--success) / 10%);
  border-color: hsl(var(--success) / 28%);
}

.chip.unknown {
  color: hsl(var(--muted-foreground));
  background: transparent;
  border-color: hsl(var(--border));
}

.chip.warn {
  color: hsl(var(--warning));
  background: hsl(var(--warning) / 12%);
  border-color: hsl(var(--warning) / 32%);
}

.chip.danger {
  color: hsl(var(--destructive));
  background: hsl(var(--destructive) / 10%);
  border-color: hsl(var(--destructive) / 28%);
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
