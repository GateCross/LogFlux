<script lang="ts" setup>
/**
 * 站点详情整页（对齐雷池防护应用，取代早期的抽屉形态）：
 * 大头部（站点名 + 域名 + 防护模式开关组 + 今日请求/拦截）
 * + 横向 Tab（基本信息 / 防护 / 高级），每 Tab 独立保存，走既有 diff 确认制。
 * 防护模式开关组写全局 simple-waf 配置（waf_policies），改动前提示对全部站点生效。
 */
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useQueryClient } from '@tanstack/vue-query';
import { message, Modal } from 'antdv-next';
import { Page } from '@vben/common-ui';

import {
  Alert,
  Button,
  Card,
  Collapse,
  CollapsePanel,
  Empty,
  Input,
  InputNumber,
  RadioButton,
  RadioGroup,
  Select,
  Space,
  Spin,
  Switch,
  TabPane,
  Tabs,
  Tag,
  Tooltip,
} from 'antdv-next';

import {
  applySimpleWafConfigApi,
  getSimpleWafConfigApi,
  type CaddySimpleWafApi,
} from '#/api/caddy/simple-waf';
import { getCaddyConfigApi } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueryKeys } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import { apiErrorMessage } from '#/utils/api-error-message';

import {
  buildCatalogSitesFromConfigPayload,
  buildConfigWorkbenchQuery,
  catalogModeLabel,
  formatMetricCount,
  normalizeSiteHost,
  siteMetricsForDomains,
} from '../catalog/service-catalog-utils';
import {
  SITE_METRICS_TODAY_MINUTES,
  useSiteMetricsQuery,
} from '../shared/composables/useSiteMetricsQuery';
import {
  ADVANCED_CAPABILITY_ITEMS,
  PROTECT_CAPABILITY_ITEMS,
  useSiteDetailEdit,
  type SiteDetailTab,
} from './use-site-detail-edit';
import SiteLogPanel from './components/SiteLogPanel.vue';

defineOptions({ name: 'CaddySiteDetail' });

const route = useRoute();
const router = useRouter();
const queryClient = useQueryClient();

// ---------- 路由参数：query 携带 host（主域名作为站点查找键） ----------
const host = computed(() => String(route.query.host ?? '').trim().toLowerCase());

// ---------- 站点定位：与目录页共享 catalog 查询缓存，不建平行数据源 ----------
const {
  data: configData,
  loading: loadingConfig,
  errorMessage: configErrorMessage,
  refetch: refetchConfig,
} = useListDetailQuery({
  queryKey: qk.caddy.catalog(),
  queryFn: () => getCaddyConfigApi(withListDetailErrorMode()),
  errorFallback: '获取站点配置失败',
});

const configRaw = computed(() => configData.value?.config ?? '');
const configHash = computed(() => configData.value?.hash ?? '');

const site = computed(() => {
  if (!host.value) return null;
  const cards = buildCatalogSitesFromConfigPayload(configData.value);
  return (
    cards.find((card) => {
      if ((card.primaryHost || '') === host.value) return true;
      return card.domains.some((d) => d.trim().toLowerCase() === host.value);
    }) ?? null
  );
});

// ---------- 今日统计：与目录页/工作台同一套共享 query（今日 24h 视角） ----------
const metricsHosts = computed(() => {
  const hosts = new Set<string>();
  for (const domain of site.value?.domains ?? [host.value]) {
    const h = normalizeSiteHost(domain);
    if (h) hosts.add(h);
  }
  return [...hosts];
});

const {
  metricsMap,
  loading: loadingMetrics,
} = useSiteMetricsQuery({
  hosts: metricsHosts,
  windowMinutes: SITE_METRICS_TODAY_MINUTES,
});

const siteMetrics = computed(() =>
  siteMetricsForDomains(metricsMap.value, site.value?.domains ?? [host.value]),
);

const todayTotal = computed(() =>
  loadingMetrics.value ? '…' : formatMetricCount(siteMetrics.value?.total),
);
const todayBlocked = computed(() =>
  loadingMetrics.value ? '…' : formatMetricCount(siteMetrics.value?.blocked),
);
const today4xx = computed(() =>
  loadingMetrics.value ? '…' : formatMetricCount(siteMetrics.value?.count4xx),
);
const today5xx = computed(() =>
  loadingMetrics.value ? '…' : formatMetricCount(siteMetrics.value?.count5xx),
);
const blockedDanger = computed(() => Number(siteMetrics.value?.blocked ?? 0) > 0);
const serverErrorDanger = computed(() => Number(siteMetrics.value?.count5xx ?? 0) > 0);
/** 4xx 用警告色而非危险色：客户端错误通常不是站点故障，与 5xx 区分开 */
const clientErrorDanger = computed(() => Number(siteMetrics.value?.count4xx ?? 0) > 0);

// ---------- 防护模式开关组：读全局 simple-waf，改动经确认后 apply（对全部站点生效） ----------
const {
  data: wafData,
  errorMessage: wafErrorMessage,
  refetch: refetchWaf,
} = useListDetailQuery({
  queryKey: [...qk.caddy.config(), 'simple-waf'],
  queryFn: () => getSimpleWafConfigApi(withListDetailErrorMode()),
  errorFallback: '获取 WAF 状态失败',
});
const wafStatus = computed(() => wafData.value ?? null);

const applyingMode = ref(false);

const wafModeValue = computed<'detectiononly' | 'off' | 'on'>(() => {
  const status = wafStatus.value;
  if (!status || !status.enabled) return 'off';
  return status.mode === 'on' ? 'on' : 'detectiononly';
});

function handleWafModeChangeEvent(event: unknown) {
  const value = (event as { target?: { value?: unknown } } | null)?.target?.value;
  handleWafModeChange(value);
}

function handleWafModeChange(value: unknown) {
  if (value !== 'on' && value !== 'detectiononly' && value !== 'off') return;
  if (value === wafModeValue.value) return;
  const status = wafStatus.value;
  if (!status) {
    message.warning('WAF 全局状态未加载，无法修改防护模式');
    return;
  }
  Modal.confirm({
    title: '防护模式将写入全局 WAF 配置',
    content:
      '此设置对全部站点生效：修改的是全局 WAF 引擎模式，应用后影响所有已挂载 WAF 的站点。确认继续？',
    okText: '确认并应用',
    cancelText: '取消',
    async onOk() {
      applyingMode.value = true;
      try {
        // 其余字段（强度/审计/请求体限制/挂载清单）保持读到的全局现状，只改模式
        const data = await applySimpleWafConfigApi({
          audit: (status.audit || 'relevantonly') as CaddySimpleWafApi.SimpleWafAudit,
          enabled: value !== 'off',
          mode: value === 'off' ? 'off' : value,
          requestBodyAccess: status.requestBodyAccess ?? true,
          requestBodyLimit: status.requestBodyLimit || 10 * 1024 * 1024,
          requestBodyNoFilesLimit: status.requestBodyNoFilesLimit || 1024 * 1024,
          siteAddresses: Array.isArray(status.siteAddresses)
            ? [...status.siteAddresses]
            : [],
          strength: (status.strength || 'balanced') as CaddySimpleWafApi.SimpleWafStrength,
          baseHash: configHash.value || undefined,
        });
        message.success(data?.message || '防护模式已应用，对全部站点生效');
        // 应用会改写运行中配置：连带失效配置/目录/指标缓存并重取 WAF 状态
        await invalidateListDetailQueryKeys(queryClient, [
          qk.caddy.config(),
          qk.caddy.catalog(),
          ['caddy', 'metrics'],
        ]);
        await refetchWaf();
        await refetchConfig();
      } catch (error) {
        message.error(apiErrorMessage(error, '应用防护模式失败'));
      } finally {
        applyingMode.value = false;
      }
    },
  });
}

// ---------- 三 Tab 编辑：解析/定点替换/diff 确认全部复用既有链路 ----------
const activeTab = ref<SiteDetailTab>('basic');

const {
  basic,
  toggles,
  basicEditable,
  capabilities,
  hasUpstreamDirective,
  logSettings,
  logLevelLabel,
  logErrorForced,
  logInlineConflict,
  logSizeMinMb,
  logSizeMaxMb,
  certOptions,
  certLoading,
  certErrorMessage,
  refetchCerts,
  selectedCertId,
  handleCertSelect,
  pendingDiff,
  saving,
  confirmError,
  blockDiff,
  changedLineCount,
  unchangedLineCount,
  requestSave,
  confirmSave,
  cancelSave,
} = useSiteDetailEdit({
  config: configRaw,
  baseHash: configHash,
  site,
  onSaved: async ({ nextHost }) => {
    // 域名变更后同步路由 host，避免详情页按旧域名查找落空
    if (nextHost) {
      await router.replace({ query: { ...route.query, host: nextHost } });
    }
    await refetchConfig();
  },
});

function goCatalog() {
  router.push({ name: 'CaddyCatalog' });
}

function openWorkbenchRaw() {
  router.push({
    name: 'CaddyConfig',
    query: buildConfigWorkbenchQuery({ mode: 'raw' }),
  });
}

/** 空态引导跳证书中心上传 */
function goCertificates() {
  router.push('/caddy/certificates');
}

/** 站点主域名：日志 Tab 按它过滤；缺失时退回路由上的查找键 */
const sitePrimaryHost = computed(() => site.value?.primaryHost || host.value);
</script>

<template>
  <Page title="站点详情" description="站点防护与上游配置的整页视图">
    <template v-if="!host">
      <Card variant="borderless">
        <Empty description="缺少 host 参数，请从站点列表进入">
          <Button @click="goCatalog">返回站点列表</Button>
        </Empty>
      </Card>
    </template>
    <template v-else>
      <Alert
        v-if="configErrorMessage"
        class="mb-4"
        type="error"
        show-icon
        :message="configErrorMessage"
      />

      <!-- 大头部：站点名 + 域名 + 防护模式开关组 + 今日统计 -->
      <Card variant="borderless" class="mb-4">
        <Spin :spinning="loadingConfig && !site">
          <div v-if="site" class="detail-header">
            <div class="header-main">
              <div class="header-titles">
                <span class="site-name">{{ site.name || '未命名站点' }}</span>
                <Tag>{{ catalogModeLabel(site.mode) }}</Tag>
              </div>
              <div class="header-domains" :title="site.domains.join('、')">
                {{ site.domains.join('、') || '未配置域名' }}
              </div>
            </div>

            <div class="header-stats">
              <div class="stat">
                <span class="stat-value">{{ todayTotal }}</span>
                <span class="stat-label">今日请求</span>
              </div>
              <div class="stat">
                <span class="stat-value" :class="{ danger: blockedDanger }">
                  {{ todayBlocked }}
                </span>
                <span class="stat-label">今日拦截</span>
              </div>
              <Tooltip title="近 24 小时客户端错误（4xx），大量出现通常意味着接口或链接失效">
                <div class="stat">
                  <span class="stat-value" :class="{ warn: clientErrorDanger }">{{ today4xx }}</span>
                  <span class="stat-label">4xx</span>
                </div>
              </Tooltip>
              <Tooltip title="近 24 小时服务端错误（5xx），通常意味着上游不可用">
                <div class="stat">
                  <span class="stat-value" :class="{ danger: serverErrorDanger }">
                    {{ today5xx }}
                  </span>
                  <span class="stat-label">5xx</span>
                </div>
              </Tooltip>
            </div>

            <div class="header-mode">
              <span class="mode-label">防护模式</span>
              <Tooltip
                :title="
                  wafErrorMessage
                    ? `WAF 状态加载失败：${wafErrorMessage}`
                    : '此设置对全部站点生效（写入全局 WAF 配置）'
                "
              >
                <RadioGroup
                  :value="wafModeValue"
                  size="small"
                  :disabled="!wafStatus || applyingMode"
                  @change="handleWafModeChangeEvent"
                >
                  <RadioButton value="on">阻断</RadioButton>
                  <RadioButton value="detectiononly">仅检测</RadioButton>
                  <RadioButton value="off">关闭</RadioButton>
                </RadioGroup>
              </Tooltip>
            </div>
          </div>
          <Empty
            v-else-if="!loadingConfig"
            description="未找到该站点，可能已被删除或域名已变更"
          >
            <Button @click="goCatalog">返回站点列表</Button>
          </Empty>
        </Spin>
      </Card>

      <!-- 横向 Tab：基本信息 / 防护 / 高级 -->
      <Card variant="borderless">
        <template v-if="site">
          <Alert
            v-if="!site.raw"
            class="mb-3"
            type="warning"
            show-icon
            message="未取到该站点的配置块原文，请刷新后重试"
          />

          <Tabs v-model:active-key="activeTab">
            <!-- 基本信息：域名增删 + 上游（地址/HTTPS/跳过验证）+ TLS 模式，必填仅域名与上游 -->
            <TabPane key="basic" tab="基本信息">
              <template v-if="basicEditable">
                <div class="field">
                  <div class="field-label">域名 / 监听地址</div>
                  <Select
                    v-model:value="basic.domains"
                    mode="tags"
                    placeholder="回车添加"
                    :token-separators="[',', ' ']"
                    :open="false"
                    class="w-full"
                  />
                </div>
                <div class="field">
                  <div class="field-label">上游地址</div>
                  <Input
                    v-model:value="basic.upstreamAddress"
                    :disabled="!hasUpstreamDirective"
                    placeholder="localhost:8080"
                  />
                  <div v-if="!hasUpstreamDirective" class="cap-note">
                    未识别到 reverse_proxy 指令，上游设置不可用
                  </div>
                </div>
                <div class="field">
                  <div class="field-label">上游 HTTPS / 跳过验证</div>
                  <Space wrap>
                    <Switch
                      v-model:checked="basic.upstreamHttps"
                      :disabled="!hasUpstreamDirective"
                      checked-children="HTTPS"
                      un-checked-children="HTTP"
                    />
                    <Tooltip
                      title="上游证书无法通过校验时跳过验证（写入 transport http { tls_insecure_skip_verify }）"
                    >
                      <Switch
                        v-model:checked="basic.upstreamSkipVerify"
                        :disabled="!hasUpstreamDirective"
                        checked-children="跳过验证"
                        un-checked-children="验证上游证书"
                      />
                    </Tooltip>
                  </Space>
                </div>
                <div class="field">
                  <div class="field-label">HTTPS（TLS 模式）</div>
                  <Tooltip
                    :title="
                      capabilities?.tls.status === 'unknown'
                        ? '无法识别当前 TLS 配置，请用原文模式调整'
                        : undefined
                    "
                  >
                    <RadioGroup
                      v-model:value="basic.tlsMode"
                      class="tls-group"
                      :disabled="capabilities?.tls.status === 'unknown'"
                    >
                      <RadioButton value="auto">自动</RadioButton>
                      <RadioButton value="dns">DNS</RadioButton>
                      <RadioButton value="internal">内部 CA</RadioButton>
                      <RadioButton value="manual">手动</RadioButton>
                      <RadioButton value="off">关闭</RadioButton>
                    </RadioGroup>
                  </Tooltip>
                  <div v-if="capabilities?.tls.status === 'unknown'" class="cap-note">
                    {{ capabilities?.tls.note }}
                  </div>
                  <template v-if="basic.tlsMode === 'manual'">
                    <Select
                      class="tls-cert-select"
                      :value="selectedCertId"
                      :options="certOptions"
                      :loading="certLoading"
                      allow-clear
                      placeholder="从证书库选择，自动填充下方路径（也可手动填写）"
                      @change="handleCertSelect"
                    >
                      <template #notFoundContent>
                        <div class="cert-empty">
                          暂无已上传证书，
                          <a @click="goCertificates">前往证书中心上传</a>
                        </div>
                      </template>
                    </Select>
                    <div v-if="certErrorMessage" class="cap-note cert-error">
                      {{ certErrorMessage }}，
                      <a @click="() => refetchCerts()">重试</a> 或直接手动填写路径
                    </div>
                    <Input
                      v-model:value="basic.manualCertPath"
                      class="tls-path-input"
                      placeholder="证书路径（容器内），如 /config/certs/1/example.com/fullchain.pem"
                    />
                    <Input
                      v-model:value="basic.manualKeyPath"
                      class="tls-path-input"
                      placeholder="私钥路径（容器内），如 /config/certs/1/example.com/privkey.pem"
                    />
                    <div class="cap-note">
                      路径须为 Caddy 容器内可读位置，可从证书中心复制存储路径
                    </div>
                  </template>
                </div>
                <div class="tab-actions">
                  <Button type="primary" @click="requestSave('basic')">
                    保存基本信息
                  </Button>
                </div>
              </template>
            </TabPane>

            <!-- 防护：per-site 挂载开关（默认继承全局）；防护模式在页头（全局生效） -->
            <TabPane key="protect" tab="防护">
              <Alert
                class="mb-3"
                type="info"
                show-icon
                message="防护模式（阻断/仅检测/关闭）在页头调整，写入全局 WAF 配置并对全部站点生效；以下开关仅控制当前站点的挂载项。"
              />
              <div
                v-for="item in PROTECT_CAPABILITY_ITEMS"
                :key="item.key"
                class="field-row"
              >
                <div>
                  <div>{{ item.label }}</div>
                  <div v-if="capabilities?.[item.key]?.status === 'unknown'" class="cap-note">
                    {{ capabilities?.[item.key]?.note }}
                  </div>
                  <div v-else class="cap-note">{{ item.hint }}</div>
                </div>
                <Tooltip
                  :title="
                    capabilities?.[item.key]?.status === 'unknown'
                      ? '无法识别当前配置，请用原文模式调整'
                      : item.hint
                  "
                >
                  <Switch
                    v-model:checked="(toggles[item.key] as boolean)"
                    :disabled="capabilities?.[item.key]?.status === 'unknown'"
                  />
                </Tooltip>
              </div>
              <div v-if="capabilities?.otherImports?.length" class="cap-note">
                其他业务片段：{{ capabilities.otherImports.join('、') }}（在原文模式维护）
              </div>
              <div class="tab-actions">
                <Button type="primary" @click="requestSave('protect')">
                  保存防护设置
                </Button>
              </div>
            </TabPane>

            <!-- 高级：日志三开关/压缩能力 + 逃生舱 raw，默认收起 -->
            <TabPane key="advanced" tab="高级">
              <Collapse>
                <CollapsePanel key="site-log" header="站点日志">
                  <Alert
                    v-if="logInlineConflict"
                    class="mb-3"
                    type="warning"
                    show-icon
                    message="检测到内联 log 指令，请在原文模式维护"
                  />
                  <div class="field-row">
                    <div>
                      <div>访问日志</div>
                      <div class="cap-note">
                        关闭后站点块不再写 import subdomain-log
                      </div>
                    </div>
                    <Switch
                      v-model:checked="logSettings.access"
                      :disabled="logInlineConflict"
                    />
                  </div>
                  <div class="field-row">
                    <div>
                      <div>错误日志</div>
                      <div class="cap-note">
                        由 log 的 level 控制；访问日志开启时会连带记录错误日志
                      </div>
                    </div>
                    <Switch
                      v-model:checked="logSettings.error"
                      :disabled="logInlineConflict || logErrorForced"
                    />
                  </div>
                  <div class="field-row">
                    <div>
                      <div>单文件上限（MB）</div>
                      <div class="cap-note">
                        写满后按 roll_size 滚动归档，保留 1 个归档文件
                      </div>
                    </div>
                    <InputNumber
                      v-model:value="logSettings.sizeMb"
                      :min="logSizeMinMb"
                      :max="logSizeMaxMb"
                      :precision="0"
                      :disabled="logInlineConflict"
                      class="log-size-input"
                    />
                  </div>
                  <div class="field-row">
                    <div>
                      <div>生效级别</div>
                      <div class="cap-note">按上方开关推导，不单独编辑</div>
                    </div>
                    <span class="cap-note">{{ logLevelLabel }}</span>
                  </div>
                </CollapsePanel>
                <CollapsePanel key="caps" header="压缩">
                  <div
                    v-for="item in ADVANCED_CAPABILITY_ITEMS"
                    :key="item.key"
                    class="field-row"
                  >
                    <div>
                      <div>{{ item.label }}</div>
                      <div
                        v-if="capabilities?.[item.key]?.status === 'unknown'"
                        class="cap-note"
                      >
                        {{ capabilities?.[item.key]?.note }}
                      </div>
                      <div v-else class="cap-note">{{ item.hint }}</div>
                    </div>
                    <Tooltip
                      :title="
                        capabilities?.[item.key]?.status === 'unknown'
                          ? '无法识别当前配置，请用原文模式调整'
                          : item.hint
                      "
                    >
                      <Switch
                        v-model:checked="(toggles[item.key] as boolean)"
                        :disabled="capabilities?.[item.key]?.status === 'unknown'"
                      />
                    </Tooltip>
                  </div>
                </CollapsePanel>
                <CollapsePanel key="raw" header="站点块原文（逃生舱）">
                  <pre class="raw-block">{{ site.raw }}</pre>
                  <div class="tab-actions">
                    <Button @click="openWorkbenchRaw">在原文模式打开</Button>
                  </div>
                </CollapsePanel>
              </Collapse>
              <div class="tab-actions">
                <Button type="primary" @click="requestSave('advanced')">
                  保存高级设置
                </Button>
              </div>
            </TabPane>

            <!-- 日志：只读查询，不参与保存；多域名站点可在面板内切换 host -->
            <TabPane key="logs" tab="日志">
              <SiteLogPanel :host="sitePrimaryHost" :domains="site.domains" />
            </TabPane>
          </Tabs>
        </template>
        <Empty
          v-else-if="!loadingConfig"
          description="未找到该站点，可能已被删除或域名已变更"
        >
          <Button @click="goCatalog">返回站点列表</Button>
        </Empty>
      </Card>
    </template>

    <!-- diff 确认弹窗：沿用抽屉语义（确认制，返回修改不丢表单状态） -->
    <Modal
      :open="!!pendingDiff"
      title="确认应用变更"
      :width="720"
      :confirm-loading="saving"
      ok-text="确认并应用"
      cancel-text="返回修改"
      @ok="confirmSave"
      @cancel="cancelSave"
    >
      <Alert
        v-if="confirmError"
        class="mb-3"
        type="error"
        show-icon
        :message="confirmError"
      />
      <div class="diff-hint">
        仅替换该站点块（块外其余 {{ unchangedLineCount }} 行不变），本次改动
        {{ changedLineCount }} 行；确认后执行「校验 → 应用 → 落库」：
      </div>
      <div class="diff-box">
        <div
          v-for="row in blockDiff"
          :key="row.key"
          class="diff-line"
          :class="`diff-${row.type}`"
        >
          <span class="diff-sign">{{
            row.type === 'added' ? '+' : row.type === 'removed' ? '-' : ' '
          }}</span>
          <span class="diff-text">{{ row.right ?? row.left }}</span>
        </div>
      </div>
    </Modal>
  </Page>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}

.mb-3 {
  margin-bottom: 12px;
}

.detail-header {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 32px;
  align-items: flex-start;
  justify-content: space-between;
}

.header-main {
  flex: 1;
  min-width: 220px;
}

.header-titles {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.site-name {
  font-size: 20px;
  font-weight: 600;
}

.header-domains {
  overflow: hidden;
  margin-top: 6px;
  font-size: 13px;
  /* muted-foreground 浅色主题下过淡，用主前景色 72% 透明度保证可读 */
  color: hsl(var(--foreground) / 72%);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-stats {
  display: flex;
  gap: 28px;
}

.stat {
  display: flex;
  flex-direction: column;
  min-width: 64px;
}

.stat-value {
  font-size: 22px;
  font-weight: 600;
  /* 等宽数字：与目录卡片、工作台一致，数值变化时不会横向抖动 */
  font-variant-numeric: tabular-nums;
  line-height: 1.3;
}

.stat-value.danger {
  color: hsl(var(--destructive));
}

/* 4xx 用警告色、5xx 用危险色：与工作台卡片同一套语义 */
.stat-value.warn {
  color: hsl(var(--warning));
}

.stat-label {
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}

.header-mode {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mode-label {
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}

.field {
  margin-bottom: 16px;
}

.field-label {
  margin-bottom: 6px;
  font-weight: 500;
}

.field-row {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid hsl(var(--border) / 60%);
}

.tab-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 16px;
}

.tls-group {
  flex-wrap: wrap;
}

.tls-cert-select {
  width: 100%;
  margin-top: 8px;
}

.cert-empty {
  padding: 8px 12px;
  color: rgba(0, 0, 0, 45%);
}

.tls-path-input {
  margin-top: 8px;
}

.cap-note {
  margin-top: 2px;
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}

.log-size-input {
  width: 160px;
}

.cap-note.cert-error {
  margin-top: 4px;
  color: #cf1322;
}

.raw-block {
  max-height: 320px;
  padding: 8px 12px;
  overflow: auto;
  font-size: 12.5px;
  line-height: 1.6;
  white-space: pre;
  background: rgba(0, 0, 0, 3%);
  border-radius: 6px;
}

.diff-hint {
  margin-bottom: 8px;
  color: rgba(0, 0, 0, 65%);
}

.diff-box {
  max-height: 420px;
  padding: 8px 0;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.6;
  background: rgba(0, 0, 0, 3%);
  border-radius: 6px;
}

.diff-line {
  display: flex;
  padding: 0 12px;
  white-space: pre;
}

.diff-sign {
  width: 18px;
  flex-shrink: 0;
  color: hsl(var(--muted-foreground));
}

/* 用主题语义变量着色，深浅色主题下都可读 */
.diff-added {
  background: hsl(var(--success) / 18%);
}

.diff-removed {
  background: hsl(var(--destructive) / 14%);
}

.diff-changed {
  background: hsl(var(--warning) / 18%);
}

.diff-same {
  color: hsl(var(--muted-foreground));
}
</style>
