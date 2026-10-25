/**
 * 站点详情页编辑状态与保存链路（自 SiteDrawer 迁移，解析层零重写）。
 *
 * - 基本/防护/高级三个 Tab 各自独立 buildNext：只收集本 Tab 的字段域改动，
 *   外科编辑复用 site-block-edit（块头/reverse_proxy/tls 定点替换）与
 *   site-capabilities（能力行定点写回），不重建全文。
 * - 保存沿用既有确认制：diff 确认 → /adapt 校验 → /load 应用 → 落库（baseHash 乐观并发）。
 * - 防护模式（阻断/仅检测/关闭）是全局 WAF 配置，不在此处编辑，由详情页头部直写 simple-waf。
 */
import { computed, reactive, ref, watch, type Ref } from 'vue';
import { message } from 'antdv-next';

import { previewCaddyConfigApi, pushCaddyConfigApi } from '#/api/caddy/server';
import { apiErrorMessage } from '#/utils/api-error-message';

import { useUploadedCertOptions } from '../certificates/use-uploaded-certs';
import type { CatalogSiteCard } from '../catalog/service-catalog-utils';
import {
  applyCapabilityToSiteBlock,
  extractSiteCapabilities,
  renderCapabilityLines,
  type SiteCapabilityKey,
  type SiteCapabilitiesSnapshot,
} from '../config/site-capabilities';
import {
  applyBasicEditsToSiteBlock,
  parseManualTlsPaths,
  parseReverseProxyTarget,
  parseUpstreamSkipVerify,
  replaceSiteBlockInConfig,
  type EditableTlsMode,
} from './site-block-edit';

export type SiteDetailTab = 'basic' | 'protect' | 'advanced';

/** 防护 Tab 的 per-site 能力开关（挂载默认继承全局，改这里只影响当前站点） */
export const PROTECT_CAPABILITY_ITEMS: Array<{
  key: SiteCapabilityKey;
  label: string;
  hint: string;
}> = [
  {
    key: 'waf',
    label: 'WAF 防护挂载',
    hint: '挂载 import waf_protect；防护模式跟随全局策略（页头可改，对全部站点生效）',
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
  {
    key: 'accessLog',
    label: '独立访问日志',
    hint: '挂载 import subdomain-log，按域名分文件',
  },
];

const ALL_CAPABILITY_ITEMS = [...PROTECT_CAPABILITY_ITEMS, ...ADVANCED_CAPABILITY_ITEMS];

/** 能力开关当前值；undefined 表示该能力状态 unknown，开关禁用 */
type CapabilityToggles = Record<SiteCapabilityKey, boolean | undefined>;

export interface SiteDetailEditOptions {
  serverId: Ref<number | undefined>;
  /** 当前服务器 Caddyfile 原文（详情页与目录页共享同一查询缓存） */
  config: Ref<string>;
  /** 当前配置哈希；确认后回传做乐观并发 */
  baseHash: Ref<string>;
  site: Ref<CatalogSiteCard | null>;
  /** 落库成功回调：刷新配置/指标，必要时同步路由 host */
  onSaved?: (payload: { scope: SiteDetailTab; nextHost?: string }) => void | Promise<void>;
}

export function useSiteDetailEdit(options: SiteDetailEditOptions) {
  const { serverId, config, baseHash, site } = options;

  // 基本信息 Tab 恒为表单编辑（雷池式低门槛，不设"复杂站点只读"分支）：
  // 域名/TLS 恒可编辑；无 reverse_proxy 时上游输入框自带禁用提示。
  const basicEditable = computed(() => Boolean(site.value?.raw));

  const capabilities = computed<SiteCapabilitiesSnapshot | null>(() => {
    if (!site.value?.raw) return null;
    return site.value.capabilities ?? extractSiteCapabilities(site.value.raw);
  });

  /** 独立访问日志的文件名：既有 subdomain-log 参数优先，否则取主域名前缀 */
  const logName = computed(() => {
    const raw = site.value?.raw ?? '';
    const match = raw.match(/^[ \t]*import\s+subdomain-log\s+(\S+)/m);
    if (match?.[1]) return match[1];
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
    waf: false,
    geo: false,
    compression: false,
    securityHeaders: false,
    accessLog: false,
    errorPage: false,
  });

  /** 站点是否存在可编辑的 reverse_proxy 指令（静态/重定向站点上游与跳过验证不可用） */
  const hasUpstreamDirective = computed(() => {
    const raw = site.value?.raw;
    if (!raw) return false;
    return Boolean(site.value?.upstream || parseReverseProxyTarget(raw));
  });

  // 从证书库选择：详情页打开且基本 Tab 可编辑时拉取，按当前服务器过滤上传证书
  const {
    options: certOptions,
    loading: certLoading,
    errorMessage: certErrorMessage,
    refetch: refetchCerts,
  } = useUploadedCertOptions(
    () => serverId.value,
    () => basicEditable.value,
  );

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
    confirmError.value = '';
  }

  watch(
    () => [site.value?.id, site.value?.raw, serverId.value],
    () => resetState(),
    { immediate: true },
  );

  /** 期望写回的 reverse_proxy 目标：地址加协议前缀；地址为空则不动 */
  function desiredUpstreamTarget(): string {
    const addr = basic.upstreamAddress.trim().replace(/^https?:\/\//i, '');
    if (!addr) return '';
    return basic.upstreamHttps ? `https://${addr}` : addr;
  }

  /**
   * 按 Tab 收集改动并生成下一站点块与整份配置；无变化返回 null。
   * 改动判定沿用抽屉语义：与现状一致的重复保存保持字节不变。
   */
  function buildNextForScope(
    scope: SiteDetailTab,
  ): { block: string; config: string; nextHost?: string } | null {
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

    if (nextBlock === raw) return null;
    const nextConfig = replaceSiteBlockInConfig(config.value, raw, nextBlock);
    if (nextConfig === config.value) return null;
    // 域名变更时带出新主域名，保存成功后同步路由查找键
    const nextHost =
      scope === 'basic' && editsDomainsChanged(current)
        ? firstValidHost(basic.domains)
        : undefined;
    return { block: nextBlock, config: nextConfig, nextHost };
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
    config: string;
    nextHost?: string;
  } | null>(null);
  const saving = ref(false);

  const unchangedLineCount = computed(() =>
    config.value.trim() ? config.value.trim().split('\n').length : 0,
  );

  function requestSave(scope: SiteDetailTab) {
    if (scope === 'basic') {
      if (basic.domains.length === 0) {
        message.warning('站点至少需要保留一个域名');
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
    const next = buildNextForScope(scope);
    if (!next) {
      message.info('没有需要保存的修改');
      return;
    }
    confirmError.value = '';
    pendingDiff.value = { scope, ...next };
  }

  async function confirmSave() {
    const next = pendingDiff.value;
    if (!next || !serverId.value) return;
    saving.value = true;
    confirmError.value = '';
    try {
      const preview = await previewCaddyConfigApi(serverId.value, {
        mode: 'raw',
        config: next.config,
        baseHash: baseHash.value || undefined,
      });
      if (preview && preview.valid === false) {
        confirmError.value = preview.errors?.[0] || '配置校验未通过';
        return;
      }
      await pushCaddyConfigApi(
        serverId.value,
        next.config,
        undefined,
        baseHash.value || undefined,
      );
      message.success('站点配置已更新');
      pendingDiff.value = null;
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
    confirmError.value = '';
  }

  return {
    basic,
    toggles,
    basicEditable,
    capabilities,
    logName,
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
    unchangedLineCount,
    requestSave,
    confirmSave,
    cancelSave,
    resetState,
  };
}
