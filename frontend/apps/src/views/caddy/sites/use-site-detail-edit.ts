/**
 * 站点详情页编辑状态与保存链路（自早期抽屉形态迁移，解析层零重写）。
 *
 * - 基本/防护/高级三个 Tab 各自独立 buildNext：只收集本 Tab 的字段域改动，
 *   外科编辑复用 site-block-edit（块头/reverse_proxy/tls 定点替换）与
 *   site-capabilities（能力行定点写回），不重建全文。
 * - 保存沿用既有确认制：diff 确认 → /adapt 校验 → /load 应用 → 落库（baseHash 乐观并发）。
 * - 防护模式与维护模式都是站点级的，均由详情页头部弹窗登记目标值后走本文件的定点改写。
 */
import { computed, reactive, ref, watch, type Ref } from 'vue';
import { message } from 'antdv-next';

import { previewCaddyConfigApi, pushCaddyConfigApi } from '#/api/caddy/server';
import { apiErrorMessage } from '#/utils/api-error-message';

import { buildLineDiff, type DiffRow } from '../config/caddy-config-diff';
import { upstreamPathError } from '../config/caddy-config-utils';
import { useUploadedCertOptions } from '../certificates/use-uploaded-certs';
import type { CatalogSiteCard } from '../catalog/service-catalog-utils';
import type { SiteProtectMode } from '../shared/protect-mode';
import {
  applyCapabilityToSiteBlock,
  applySiteLogToSiteBlock,
  applyWafModeToSiteBlock,
  extractSiteCapabilities,
  renderCapabilityLines,
  type SiteCapabilityKey,
  type SiteCapabilitiesSnapshot,
} from '../config/site-capabilities';
import {
  DEFAULT_SITE_LOG_SETTINGS,
  describeSiteLogLevel,
  isSameSiteLogSettings,
  normalizeSiteLogSizeMb,
  parseSiteLogImport,
  readSiteLogSettings,
  settingsToLevel,
  SITE_LOG_SIZE_MAX_MB,
  SITE_LOG_SIZE_MIN_MB,
  validateSiteLogSettings,
  type SiteLogSettings,
} from '../config/site-log-settings';
import {
  applyBasicEditsToSiteBlock,
  parseManualTlsPaths,
  parseReverseProxyTarget,
  parseUpstreamSkipVerify,
  replaceSiteBlockInConfig,
  type EditableTlsMode,
} from './site-block-edit';

export type SiteDetailTab = 'basic' | 'protect' | 'advanced' | 'logs';

/** 防护 Tab 的 per-site 能力开关（挂载默认继承全局，改这里只影响当前站点） */
export const PROTECT_CAPABILITY_ITEMS: Array<{
  key: SiteCapabilityKey;
  label: string;
  hint: string;
}> = [
  {
    key: 'waf',
    label: 'WAF 防护挂载',
    hint: '挂载 import waf_protect；具体模式（防护/观察）在页头切换，只影响本站点',
  },
  {
    key: 'geo',
    label: 'IP 区域限制',
    hint: '挂载 import geo_auth，国家/地区白名单在「访问控制」配置',
  },
  {
    key: 'securityHeaders',
    label: '安全响应头',
    hint: '挂载 import security_headers 片段',
  },
  {
    key: 'errorPage',
    label: '403 错误页跳转',
    hint: '挂载 import deny_403_redirect 片段',
  },
];

/** 高级 Tab 的能力开关：默认收起，按需覆盖 */
export const ADVANCED_CAPABILITY_ITEMS: Array<{
  key: SiteCapabilityKey;
  label: string;
  hint: string;
}> = [
  {
    key: 'compression',
    label: '压缩',
    hint: '挂载 import compression（zstd/gzip）',
  },
];

const ALL_CAPABILITY_ITEMS = [...PROTECT_CAPABILITY_ITEMS, ...ADVANCED_CAPABILITY_ITEMS];

/**
 * 站点块里是否存在独立于 snippet 的内联 log 指令。
 *
 * 三开关只写 import subdomain-log，管不到用户手写的 log { ... }；两者同时存在时
 * 后写的一条会覆盖前一条，所以这里探到内联指令就让 UI 禁用并引导去原文模式。
 */
function hasInlineSiteLogDirective(siteBlock: string): boolean {
  for (const line of (siteBlock ?? '').split('\n')) {
    const trimmed = line.replace(/\r$/, '').trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    if (/^log(?:\s*\{|\s*$)/.test(trimmed)) return true;
  }
  return false;
}

/** 能力开关当前值；undefined 表示该能力状态 unknown，开关禁用 */
type CapabilityToggles = Record<SiteCapabilityKey, boolean | undefined>;

export interface SiteDetailEditOptions {
  /** 当前 Caddyfile 原文（详情页与目录页共享同一查询缓存） */
  config: Ref<string>;
  /** 当前配置哈希；确认后回传做乐观并发 */
  baseHash: Ref<string>;
  site: Ref<CatalogSiteCard | null>;
  /** 落库成功回调：刷新配置/指标，必要时同步路由 host */
  onSaved?: (payload: { scope: SiteDetailTab; nextHost?: string }) => void | Promise<void>;
}

export function useSiteDetailEdit(options: SiteDetailEditOptions) {
  const { config, baseHash, site } = options;

  // 基本信息 Tab 恒为表单编辑（雷池式低门槛，不设"复杂站点只读"分支）：
  // 域名/TLS 恒可编辑；无 reverse_proxy 时上游输入框自带禁用提示。
  const basicEditable = computed(() => Boolean(site.value?.raw));

  const capabilities = computed<SiteCapabilitiesSnapshot | null>(() => {
    if (!site.value?.raw) return null;
    return site.value.capabilities ?? extractSiteCapabilities(site.value.raw);
  });

  /** 独立访问日志的文件名：既有 import 的文件名参数优先，否则取主域名前缀 */
  const logName = computed(() => {
    const raw = site.value?.raw ?? '';
    const parsed = parseSiteLogImport(raw);
    if (parsed?.name) return parsed.name;
    const host = site.value?.primaryHost ?? '';
    return host.split('.')[0] || 'site';
  });

  const basic = reactive({
    domains: [] as string[],
    /** 上游地址（不含协议前缀；协议走 upstreamHttps 开关表达） */
    upstreamAddress: '',
    upstreamHttps: false,
    upstreamSkipVerify: false,
    tlsMode: 'auto' as EditableTlsMode,
    /** manual 模式的证书/私钥路径（Caddy 容器内路径，可从证书库选择回填） */
    manualCertPath: '',
    manualKeyPath: '',
  });

  /** diff 确认前的校验错误；声明需在 resetState 之前（immediate watch 会先执行它） */
  const confirmError = ref('');

  const toggles = reactive<CapabilityToggles>({
    maintenance: false,
    waf: false,
    geo: false,
    compression: false,
    securityHeaders: false,
    accessLog: false,
    errorPage: false,
  });

  // ---------- 「日志」分组：三开关 + 单文件上限，读写都落在站点块 import 参数上 ----------

  /** 日志表单态；由站点块里的 import subdomain-log 回填，保存时再写回 */
  const logSettings = reactive<SiteLogSettings>({ ...DEFAULT_SITE_LOG_SETTINGS });

  /** 站点块里另有内联 log 指令：三开关管不到它，禁用并提示去原文模式 */
  const logInlineConflict = computed(() =>
    hasInlineSiteLogDirective(site.value?.raw ?? ''),
  );

  /** 生效 level 只读展示：INFO 含访问+错误，ERROR 只留错误，OFF 表示整条 import 不写 */
  const logLevel = computed(() => settingsToLevel(logSettings));

  /** 生效档位的中文说明，配合只读 level 一起展示 */
  const logLevelLabel = computed(() =>
    describeSiteLogLevel({
      access: logSettings.access,
      error: logSettings.error,
      sizeMb: normalizeSiteLogSizeMb(logSettings.sizeMb),
    }),
  );

  /** 错误日志开关是否受访问日志钳制（开访问必然连带错误，无权单独关闭） */
  const logErrorForced = computed(() => logSettings.access);

  // level 是最低级别而非白名单：开启访问日志时强制把错误日志拉回开启，
  // 避免 UI 停留在 Caddy 无法表达的「开访问 + 关错误」组合上。
  watch(
    () => logSettings.access,
    (access) => {
      if (access) logSettings.error = true;
    },
  );

  function normalizeLogSettings(): SiteLogSettings {
    return {
      access: logSettings.access,
      error: logSettings.error,
      sizeMb: normalizeSiteLogSizeMb(logSettings.sizeMb),
    };
  }

  /** 站点是否存在可编辑的 reverse_proxy 指令（静态/重定向站点上游与跳过验证不可用） */
  const hasUpstreamDirective = computed(() => {
    const raw = site.value?.raw;
    if (!raw) return false;
    return Boolean(site.value?.upstream || parseReverseProxyTarget(raw));
  });

  // 从证书库选择：详情页打开且基本 Tab 可编辑时拉取上传证书
  const {
    options: certOptions,
    loading: certLoading,
    errorMessage: certErrorMessage,
    refetch: refetchCerts,
  } = useUploadedCertOptions(() => basicEditable.value);

  /** 当前选中的证书记录 id；清空选择不动已填路径，仍允许手动输入 */
  const selectedCertId = ref<number | undefined>();
  /** 既有 manual 双路径与某上传证书一致时回显选中（不清除用户已做的选择） */
  function syncCertSelection() {
    if (selectedCertId.value != null) return;
    const cert = basic.manualCertPath.trim();
    const key = basic.manualKeyPath.trim();
    if (!cert || !key) return;
    selectedCertId.value = certOptions.value.find(
      (item) => item.certPath === cert && item.keyPath === key,
    )?.value;
  }

  // 选项异步就绪时补一次回显；切换站点由 resetState 同步触发
  watch(certOptions, syncCertSelection);

  function handleCertSelect(value: unknown) {
    const option = certOptions.value.find((item) => item.value === value);
    selectedCertId.value = option?.value;
    if (option) {
      basic.manualCertPath = option.certPath;
      basic.manualKeyPath = option.keyPath;
    }
  }

  function resetState() {
    const current = site.value;
    const caps = capabilities.value;
    basic.domains = [...(current?.domains ?? [])];
    // 目标里的协议前缀拆到 upstreamHttps 开关，输入框只留 地址:端口
    const target = current?.upstream || (current?.raw ? parseReverseProxyTarget(current.raw) : '') || '';
    basic.upstreamAddress = target.replace(/^https?:\/\//i, '');
    basic.upstreamHttps = /^https:\/\//i.test(target);
    basic.upstreamSkipVerify = current?.raw ? parseUpstreamSkipVerify(current.raw) : false;
    basic.tlsMode = caps?.tls.mode ?? 'auto';
    const manualPaths =
      caps?.tls.mode === 'manual' ? parseManualTlsPaths(current?.raw ?? '') : null;
    basic.manualCertPath = manualPaths?.certPath ?? '';
    basic.manualKeyPath = manualPaths?.keyPath ?? '';
    selectedCertId.value = undefined;
    syncCertSelection();
    for (const item of ALL_CAPABILITY_ITEMS) {
      const status = caps?.[item.key]?.status;
      toggles[item.key] = status === 'unknown' ? undefined : status === 'on';
    }
    // 日志三开关从站点块 import 参数回填；块内没有 import 即两开关全关
    const parsedLog = readSiteLogSettings(current?.raw ?? '');
    logSettings.access = parsedLog.settings.access;
    logSettings.error = parsedLog.settings.error;
    logSettings.sizeMb = parsedLog.settings.sizeMb;
    confirmError.value = '';
  }

  watch(
    () => [site.value?.id, site.value?.raw],
    () => resetState(),
    { immediate: true },
  );

  /** 期望写回的 reverse_proxy 目标：地址加协议前缀；地址为空则不动 */
  function desiredUpstreamTarget(): string {
    const addr = basic.upstreamAddress.trim().replace(/^https?:\/\//i, '');
    if (!addr) return '';
    return basic.upstreamHttps ? `https://${addr}` : addr;
  }

  /** 站点级防护模式的目标值；非空时由 buildNextForScope 一并写回 */
  const pendingProtectMode = ref<SiteProtectMode | null>(null);

  /**
   * 请求切换站点防护模式：只登记目标值并打开 diff 确认，
   * 真正的站点块改写发生在 buildNextForScope，与其它能力同一条链路。
   */
  function requestProtectMode(mode: SiteProtectMode) {
    pendingProtectMode.value = mode;
    requestSave('protect');
  }

  /**
   * 按 Tab 收集改动并生成下一站点块与整份配置；无变化返回 null。
   * 改动判定沿用抽屉语义：与现状一致的重复保存保持字节不变。
   */
  function buildNextForScope(
    scope: SiteDetailTab,
  ): {
    block: string;
    /** 改动前的站点块原文，供 diff 弹窗渲染删除行 */
    previousBlock: string;
    config: string;
    nextHost?: string;
  } | null {
    const current = site.value;
    const raw = current?.raw;
    if (!current || !raw) return null;

    let nextBlock = raw;

    if (scope === 'basic' && basicEditable.value) {
      const edits: Parameters<typeof applyBasicEditsToSiteBlock>[1] = {};
      if (basic.domains.join(' ') !== current.domains.join(' ')) {
        edits.domains = basic.domains;
      }
      const currentTarget = current.upstream || parseReverseProxyTarget(raw) || '';
      const desired = desiredUpstreamTarget();
      if (desired && desired !== currentTarget) {
        edits.upstream = desired;
      }
      if (basic.upstreamSkipVerify !== parseUpstreamSkipVerify(raw)) {
        edits.upstreamSkipVerify = basic.upstreamSkipVerify;
      }
      const currentTls = capabilities.value?.tls.mode ?? 'auto';
      if (basic.tlsMode === 'manual') {
        const cert = basic.manualCertPath.trim();
        const key = basic.manualKeyPath.trim();
        const currentPaths = currentTls === 'manual' ? parseManualTlsPaths(raw) : null;
        const changed =
          !currentPaths ||
          currentPaths.certPath !== cert ||
          currentPaths.keyPath !== key;
        // 路径齐全且相对现状有变化才写：与现状一致的重复保存保持字节不变
        if (cert && key && changed) {
          edits.tlsMode = 'manual';
          edits.tlsCertPath = cert;
          edits.tlsKeyPath = key;
        }
      } else if (basic.tlsMode !== currentTls) {
        edits.tlsMode = basic.tlsMode;
      }
      if (Object.keys(edits).length > 0) {
        nextBlock = applyBasicEditsToSiteBlock(nextBlock, edits);
      }
    }

    const keys =
      scope === 'protect'
        ? PROTECT_CAPABILITY_ITEMS.map((item) => item.key)
        : scope === 'advanced'
          ? ADVANCED_CAPABILITY_ITEMS.map((item) => item.key)
          : [];
    for (const key of keys) {
      const caps = capabilities.value;
      const currentStatus = caps?.[key]?.status;
      const target = toggles[key];
      if (target === undefined || currentStatus === 'unknown') continue;
      if (target === (currentStatus === 'on')) continue;
      nextBlock = applyCapabilityToSiteBlock(
        nextBlock,
        key,
        renderCapabilityLines(key, { enabled: target, name: logName.value }),
      );
    }

    // 日志分组：三开关与现状不同才写回，相同则保持字节不变（重复保存无 diff）
    if (scope === 'advanced' && !logInlineConflict.value) {
      const desiredLog = normalizeLogSettings();
      const currentLog = readSiteLogSettings(nextBlock).settings;
      if (!isSameSiteLogSettings(desiredLog, currentLog)) {
        nextBlock = applySiteLogToSiteBlock(nextBlock, desiredLog, logName.value);
      }
    }

    // 站点级防护模式：与维护模式同一条定点改写链路，只换站点块内的那一处
    if (scope === 'protect' && pendingProtectMode.value) {
      const mode = pendingProtectMode.value;
      if (mode === 'maintenance') {
        nextBlock = applyCapabilityToSiteBlock(
          nextBlock,
          'maintenance',
          renderCapabilityLines('maintenance', { enabled: true }),
        );
        // 维护模式接管整站，WAF 挂载同时摘掉，避免 503 之外还跑一套规则
        nextBlock = applyWafModeToSiteBlock(nextBlock, 'off');
      } else {
        // 切到其它模式时先撤掉维护配置，再换 WAF 挂载行
        nextBlock = applyCapabilityToSiteBlock(nextBlock, 'maintenance', []);
        nextBlock = applyWafModeToSiteBlock(
          nextBlock,
          mode === 'off' ? 'off' : mode,
        );
      }
    }

    if (nextBlock === raw) return null;
    const nextConfig = replaceSiteBlockInConfig(config.value, raw, nextBlock);
    if (nextConfig === config.value) return null;
    // 域名变更时带出新主域名，保存成功后同步路由查找键
    const nextHost =
      scope === 'basic' && editsDomainsChanged(current)
        ? firstValidHost(basic.domains)
        : undefined;
    return { block: nextBlock, previousBlock: raw, config: nextConfig, nextHost };
  }

  function editsDomainsChanged(current: CatalogSiteCard): boolean {
    return basic.domains.join(' ') !== current.domains.join(' ');
  }

  function firstValidHost(domains: string[]): string | undefined {
    for (const domain of domains) {
      const host = domain.trim().toLowerCase();
      // 与 normalizeSiteHost 语义保持一致的轻量判断：纯监听地址不可作查找键
      if (host && !host.startsWith(':')) return host;
    }
    return undefined;
  }

  // ---------- diff 确认制保存（与抽屉一致：确认 → 校验 → 应用 → 落库） ----------

  const pendingDiff = ref<{
    scope: SiteDetailTab;
    block: string;
    /** 改动前的站点块原文，供 diff 弹窗渲染删除行 */
    previousBlock: string;
    config: string;
    nextHost?: string;
  } | null>(null);
  const saving = ref(false);

  /**
   * 变更预览：按「旧块 → 新块」做行级 diff。
   * 只展示真正变动的行，未变化的行既不渲染也不计数，
   * 避免旧实现把整块当新增、把整份配置行数当成"未变行数"。
   */
  const blockDiff = computed(() => {
    const pending = pendingDiff.value;
    if (!pending) return [] as DiffRow[];
    return buildLineDiff(pending.previousBlock, pending.block);
  });

  /** 本次真正改动的行数（新增 + 删除 + 修改），用于弹窗摘要 */
  const changedLineCount = computed(
    () => blockDiff.value.filter((row) => row.type !== 'same').length,
  );

  /** 该站点块之外、本次完全不动的配置行数 */
  const unchangedLineCount = computed(() => {
    const pending = pendingDiff.value;
    if (!pending) return 0;
    const total = config.value.trim() ? config.value.trim().split('\n').length : 0;
    const previous = pending.previousBlock.trim()
      ? pending.previousBlock.trim().split('\n').length
      : 0;
    return Math.max(0, total - previous);
  });

  function requestSave(scope: SiteDetailTab) {
    if (scope === 'basic') {
      if (basic.domains.length === 0) {
        message.warning('站点至少需要保留一个域名');
        return;
      }
      const upstreamError = upstreamPathError(basic.upstreamAddress);
      if (upstreamError) {
        message.warning(upstreamError);
        return;
      }
      if (
        basic.tlsMode === 'manual' &&
        (!basic.manualCertPath.trim() || !basic.manualKeyPath.trim())
      ) {
        message.warning('手动证书模式需同时填写证书与私钥路径');
        return;
      }
    }
    if (scope === 'advanced' && !logInlineConflict.value) {
      // 大小上限必须是 1..1024 的整数；「开访问 + 关错误」由 UI 强制拉平，不会出现在这里
      const problems = validateSiteLogSettings(normalizeLogSettings());
      if (problems.length > 0) {
        message.warning(problems[0]!);
        return;
      }
    }
    const next = buildNextForScope(scope);
    if (!next) {
      pendingProtectMode.value = null;
      message.info('没有需要保存的修改');
      return;
    }
    confirmError.value = '';
    pendingDiff.value = { scope, ...next };
  }

  async function confirmSave() {
    const next = pendingDiff.value;
    if (!next) return;
    saving.value = true;
    confirmError.value = '';
    try {
      const preview = await previewCaddyConfigApi({
        mode: 'raw',
        config: next.config,
        baseHash: baseHash.value || undefined,
      });
      if (preview && preview.valid === false) {
        confirmError.value = preview.errors?.[0] || '配置校验未通过';
        return;
      }
      await pushCaddyConfigApi(
        next.config,
        undefined,
        baseHash.value || undefined,
      );
      message.success('站点配置已更新');
      pendingDiff.value = null;
      pendingProtectMode.value = null;
      await options.onSaved?.({ scope: next.scope, nextHost: next.nextHost });
    } catch (error) {
      confirmError.value = apiErrorMessage(error, '保存失败');
    } finally {
      saving.value = false;
    }
  }

  /** 返回修改：不丢弃表单状态，仅撤下 diff 确认 */
  function cancelSave() {
    pendingDiff.value = null;
    pendingProtectMode.value = null;
    confirmError.value = '';
  }

  return {
    basic,
    toggles,
    basicEditable,
    capabilities,
    logName,
    logSettings,
    logLevel,
    logLevelLabel,
    logErrorForced,
    logInlineConflict,
    logSizeMinMb: SITE_LOG_SIZE_MIN_MB,
    logSizeMaxMb: SITE_LOG_SIZE_MAX_MB,
    hasUpstreamDirective,
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
    requestProtectMode,
    confirmSave,
    cancelSave,
    resetState,
  };
}
