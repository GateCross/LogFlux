/** 配置加载/预览/保存与 Blocks/Raw 草稿 */

import { computed, reactive, ref, type Ref } from 'vue';
import { message } from 'antdv-next';
import { useQueryClient } from '@tanstack/vue-query';

import {
  discoverDockerServicesApi,
  getCaddyConfigApi,
  previewCaddyConfigApi,
  pushCaddyConfigApi,
} from '#/api/caddy/server';
import type { DockerDiscoveryCandidate } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueryKeys } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { apiErrorMessage } from '#/utils/api-error-message';

import { parseCaddyfileToBlocks, buildCaddyfileFromBlocks } from '../caddy-config-blocks';
import {
  formatCaddyfile,
  genId,
  normalizeModules,
  validateStructuredConfig,
} from '../caddy-config-utils';
import {
  buildQuickConfigState,
  createQuickSiteDraft,
  mergeQuickConfigDrafts,
  type QuickSiteDraft,
} from '../quick-config-utils';
import type { SiteWizardWafDraft } from '../site-wizard-utils';
import type { DockerDiscoveryCandidate as SessionDockerCandidate } from '../docker-discovery-utils';
import type {
  CaddyBlockDraft,
  CaddyFormModel,
  CaddyPageMode,
  HealthCheck,
  PreservedCaddyBlock,
} from '../types';

export type SavePreviewKind = 'blocks' | 'raw';

export function createEmptyFormModel(): CaddyFormModel {
  return {
    schemaVersion: 1,
    global: { raw: '' },
    upstreams: [],
    sites: [],
  };
}

export function defaultHealthCheck(): HealthCheck {
  return {
    path: '/health',
    interval: '10s',
    timeout: '5s',
  };
}

export function blockKindLabel(kind: PreservedCaddyBlock['kind']) {
  if (kind === 'global') return '全局';
  if (kind === 'snippet') return 'snippet';
  if (kind === 'site') return 'site';
  return '未知';
}

export function blockKindColor(kind: PreservedCaddyBlock['kind']) {
  if (kind === 'global') return 'green';
  if (kind === 'snippet') return 'blue';
  if (kind === 'site') return 'orange';
  return 'default';
}

export const modeOptions = [
  { label: '分块配置', value: 'blocks' },
  { label: '防火墙', value: 'waf' },
  { label: '原始配置', value: 'raw' },
  { label: '预览', value: 'preview' },
];

export const quickModeOptions = [
  { label: '反向代理', value: 'reverse_proxy' },
  { label: '静态文件', value: 'file_server' },
  { label: '重定向', value: 'redirect' },
];

export const tlsModeOptions = [
  { label: '自动 HTTPS', value: 'auto' },
  { label: '内部证书', value: 'internal' },
  { label: '关闭 TLS', value: 'off' },
];

export const lbPolicyOptions = [
  { label: '轮询', value: 'round_robin' },
  { label: '最少连接', value: 'least_conn' },
  { label: 'IP Hash', value: 'ip_hash' },
];

export interface UseCaddyConfigIOOptions {
  selectedServerId: Ref<number | undefined>;
  /** 服务目录等入口深链 mode */
  resolveRouteMode?: () => CaddyPageMode | undefined;
  /** 切换/加载配置时重置站点指标缓存 */
  onConfigReset?: () => void;
  /** 配置/草稿就绪后 debounce 拉取指标 */
  onConfigReady?: () => void;
}

export function useCaddyConfigIO(options: UseCaddyConfigIOOptions) {
  const { selectedServerId, resolveRouteMode, onConfigReset, onConfigReady } = options;
  const queryClient = useQueryClient();

  const loadingConfig = ref(false);
  /** 每次加载递增，避免旧服务器的慢响应覆盖当前草稿。 */
  let configRequestVersion = 0;
  const configErrorMessage = ref<string | null>(null);
  const saving = ref(false);
  const previewing = ref(false);

  const configContent = ref('');
  const mode = ref<CaddyPageMode>('blocks');
  const lastEditMode = ref<'blocks' | 'raw'>('blocks');
  const structuredAvailable = ref(false);
  const formModel = ref<CaddyFormModel>(createEmptyFormModel());
  const preservedBlocks = ref<PreservedCaddyBlock[]>([]);
  const quickSiteDrafts = ref<QuickSiteDraft[]>([]);
  const complexSiteSummaries = ref<
    Array<{ domains: string[]; id: string; name: string; reasons: string[] }>
  >([]);
  const activeQuickSiteId = ref<string>();

  const savePreview = reactive<{
    actions: string[];
    config: string;
    errors: string[];
    kind: SavePreviewKind;
    modules: string;
    open: boolean;
    original: string;
  }>({
    actions: [],
    config: '',
    errors: [],
    kind: 'blocks',
    modules: '',
    open: false,
    original: '',
  });

  /** 站点创建向导：默认仅产出草稿，不自动 /load */
  const siteWizardOpen = ref(false);
  /** 向导携带的可选 WAF 意图（草稿阶段不调用 WAF 接口） */
  const pendingWizardWaf = ref<SiteWizardWafDraft | null>(null);
  /** 向导本地预览用：合并草稿后的 Caddyfile 文本（未 /load） */
  const wizardLocalPreview = ref('');

  /** Docker 发现：会话候选（内存，不建 discovery DB） */
  const dockerDiscoveryOpen = ref(false);
  const dockerDiscoveryScanning = ref(false);
  const dockerDiscoveryResult = ref<{
    list: SessionDockerCandidate[];
    scannedAt: string;
    message: string;
  } | null>(null);

  function applyRouteModeDeepLink() {
    const next = resolveRouteMode?.();
    if (!next) return;
    mode.value = next;
    if (next === 'blocks' || next === 'raw') {
      lastEditMode.value = next;
    }
  }

  /** 上游池下拉选项（用于 Simple reverse_proxy 选择池名） */
  const upstreamPoolOptions = computed(() =>
    formModel.value.upstreams
      .map((item) => item.name?.trim())
      .filter((name): name is string => Boolean(name))
      .map((name) => ({ label: `上游池：${name}`, value: name })),
  );

  const activeQuickSite = computed(() =>
    quickSiteDrafts.value.find((site) => site.id === activeQuickSiteId.value),
  );

  const globalPreservedBlocks = computed(() =>
    preservedBlocks.value
      .map((block, index) => ({ block, index }))
      .filter(({ block }) => block.kind !== 'site'),
  );

  const preservedSiteBlocks = computed(() =>
    preservedBlocks.value
      .map((block, index) => ({ block, index }))
      .filter(({ block }) => block.kind === 'site'),
  );

  const mergedQuickFormModel = computed(() =>
    mergeQuickConfigDrafts(formModel.value, quickSiteDrafts.value),
  );

  const hasPreservedContent = computed(() =>
    preservedBlocks.value.some((block) => block.raw.trim().length > 0),
  );

  const generatedConfig = computed(() => {
    const draft: CaddyBlockDraft = {
      ...mergedQuickFormModel.value,
      preservedBlocks: preservedBlocks.value,
    };
    return buildCaddyfileFromBlocks(draft, { sourceOrder: configContent.value });
  });

  const previewConfig = computed(() =>
    formatCaddyfile(
      lastEditMode.value === 'raw'
        ? configContent.value
        : generatedConfig.value.trim()
          ? generatedConfig.value
          : configContent.value,
    ),
  );

  const quickValidationErrors = computed(() =>
    validateStructuredConfig(mergedQuickFormModel.value, hasPreservedContent.value),
  );

  function syncQuickStateFromForm(model: CaddyFormModel) {
    const { simpleSites, complexSites } = buildQuickConfigState(model);
    quickSiteDrafts.value = simpleSites;
    complexSiteSummaries.value = complexSites;
    activeQuickSiteId.value = simpleSites.some((site) => site.id === activeQuickSiteId.value)
      ? activeQuickSiteId.value
      : simpleSites[0]?.id;
  }

  function applyDraft(draft: CaddyBlockDraft) {
    formModel.value = {
      schemaVersion: draft.schemaVersion,
      global: draft.global,
      upstreams: draft.upstreams,
      sites: draft.sites,
    };
    preservedBlocks.value = draft.preservedBlocks ?? [];
    structuredAvailable.value = true;
    syncQuickStateFromForm(formModel.value);
    lastEditMode.value = 'blocks';
    mode.value = 'blocks';
  }

  function hasFormContent(model: CaddyFormModel) {
    return Boolean(
      model.sites?.length || model.upstreams?.length || model.global?.raw?.trim(),
    );
  }

  function hasPreservedBlocks(blocks: PreservedCaddyBlock[]) {
    return blocks.some((block) => block.raw.trim().length > 0);
  }

  function parseRawToBlocks(showSuccess = false) {
    if (!configContent.value.trim()) {
      message.warning('原始配置为空，无法解析');
      return false;
    }
    const parsed = parseCaddyfileToBlocks(configContent.value);
    if (!parsed.sites.length && !parsed.global?.raw && !parsed.preservedBlocks.length) {
      message.error('未解析到可用结构化配置');
      return false;
    }
    applyDraft(parsed);
    if (showSuccess) message.success('已从原始配置解析');
    return true;
  }

  function loadModules(modules: string | undefined) {
    if (!modules) return false;
    try {
      const parsed = JSON.parse(modules);
      const normalized = normalizeModules(parsed);
      const modulePreservedBlocks = Array.isArray(parsed?.preservedBlocks)
        ? parsed.preservedBlocks
        : [];
      if (!hasFormContent(normalized) && !hasPreservedBlocks(modulePreservedBlocks)) {
        return false;
      }
      formModel.value = normalized;
      preservedBlocks.value = modulePreservedBlocks;
      structuredAvailable.value = true;
      syncQuickStateFromForm(formModel.value);
      return true;
    } catch {
      message.warning('结构化配置解析失败，已回退到原始配置解析');
      return false;
    }
  }

  function mergePreservedBlocksFromRaw(parsed: CaddyBlockDraft) {
    const siteDomains = new Set(formModel.value.sites.flatMap((site) => site.domains));
    const existingRawSet = new Set(
      preservedBlocks.value.map((block) => block.raw.trim()).filter(Boolean),
    );
    const nextBlocks = [...preservedBlocks.value];

    for (const block of parsed.preservedBlocks ?? []) {
      const raw = block.raw.trim();
      if (!raw || existingRawSet.has(raw)) continue;
      if (block.kind === 'site') {
        const firstDomain = block.title.split(/[\s,]+/)[0];
        if (firstDomain && siteDomains.has(firstDomain)) continue;
      }
      nextBlocks.push(block);
      existingRawSet.add(raw);
    }

    preservedBlocks.value = nextBlocks;
  }

  async function fetchConfig() {
    const serverId = selectedServerId.value;
    if (!serverId) return;
    const requestVersion = ++configRequestVersion;
    configErrorMessage.value = null;
    loadingConfig.value = true;
    try {
      const data = await queryClient.fetchQuery({
        queryKey: qk.caddy.config(serverId),
        queryFn: () =>
          getCaddyConfigApi(serverId, withListDetailErrorMode()),
      });
      // 服务器已经切换或有更新的加载请求时，丢弃过期响应。
      if (
        requestVersion !== configRequestVersion ||
        selectedServerId.value !== serverId
      ) {
        return;
      }
      // 远程权威快照 → seed 本地草稿（configContent / formModel / quickSiteDrafts）
      // 草稿本身不写回 query cache（9.2 硬边界）
      configContent.value = data?.config ?? '';
      formModel.value = createEmptyFormModel();
      preservedBlocks.value = [];
      quickSiteDrafts.value = [];
      complexSiteSummaries.value = [];
      activeQuickSiteId.value = undefined;
      structuredAvailable.value = false;
      // 切换配置时重置指标缓存（随后 debounce 重新拉取）
      onConfigReset?.();

      const loadedModules = loadModules(typeof data?.modules === 'string' ? data.modules : undefined);
      if (configContent.value.trim()) {
        const parsed = parseCaddyfileToBlocks(configContent.value);
        if (loadedModules) {
          mergePreservedBlocksFromRaw(parsed);
          applyRouteModeDeepLink();
          onConfigReady?.();
          return;
        } else if (
          parsed.sites.length > 0 ||
          parsed.global?.raw ||
          parsed.preservedBlocks.length > 0
        ) {
          applyDraft(parsed);
          // 服务目录深链 mode 优先于 applyDraft 默认的 blocks
          applyRouteModeDeepLink();
          onConfigReady?.();
          return;
        }
      }

      syncQuickStateFromForm(formModel.value);
      mode.value = 'blocks';
      lastEditMode.value = 'blocks';
      // 深链 mode（如 waf）覆盖默认 blocks
      applyRouteModeDeepLink();
      onConfigReady?.();
    } catch (error) {
      if (
        requestVersion === configRequestVersion &&
        selectedServerId.value === serverId
      ) {
        // 页内 Alert 展示错误
        configErrorMessage.value = apiErrorMessage(error, '获取配置失败');
      }
    } finally {
      if (requestVersion === configRequestVersion) {
        loadingConfig.value = false;
      }
    }
  }

  function addQuickSite() {
    const site = createQuickSiteDraft({
      id: genId(),
      name: `新站点-${quickSiteDrafts.value.length + 1}`,
      domains: [],
      mode: 'reverse_proxy',
      upstream: 'localhost:8080',
      lbPolicy: 'round_robin',
    });
    quickSiteDrafts.value.push(site);
    activeQuickSiteId.value = site.id;
  }

  /** 打开站点创建向导（域名 → 上游 → TLS → 可选 WAF → Preview → Apply） */
  function openSiteWizard() {
    wizardLocalPreview.value = '';
    pendingWizardWaf.value = null;
    siteWizardOpen.value = true;
  }

  /** 打开 Docker 发现（label 扫描 → 会话候选；不落库、不 /load） */
  function openDockerDiscovery() {
    dockerDiscoveryOpen.value = true;
  }

  /**
   * 扫描 Docker labels（只读）。结果仅存会话 ref，不写平行 discovery DB。
   */
  async function handleDockerDiscoveryScan() {
    dockerDiscoveryScanning.value = true;
    try {
      const data = await discoverDockerServicesApi();
      const list = (data?.list ?? []) as DockerDiscoveryCandidate[];
      dockerDiscoveryResult.value = {
        list: list as SessionDockerCandidate[],
        scannedAt: data?.scannedAt || '',
        message: data?.message || '',
      };
      if (!list.length) {
        message.info(data?.message || '未发现带 logflux.enable 标签的容器');
      } else {
        message.success(data?.message || `发现 ${list.length} 个候选（会话草稿）`);
      }
    } catch (error) {
      message.error(apiErrorMessage(error, 'Docker 发现扫描失败'));
    } finally {
      dockerDiscoveryScanning.value = false;
    }
  }

  /**
   * 将向导草稿合并进 quickSiteDrafts（纯本地，不调用任何 API /load）。
   */
  function mergeWizardDraftIntoWorkbench(
    draft: QuickSiteDraft,
    meta?: { waf?: SiteWizardWafDraft },
  ) {
    const next: QuickSiteDraft = {
      ...draft,
      id: draft.id || genId(),
      domains: [...(draft.domains ?? [])],
      healthCheck: draft.healthCheck
        ? {
            path: draft.healthCheck.path,
            interval: draft.healthCheck.interval,
            timeout: draft.healthCheck.timeout,
          }
        : undefined,
    };
    const idx = quickSiteDrafts.value.findIndex((s) => s.id === next.id);
    if (idx >= 0) {
      quickSiteDrafts.value[idx] = next;
    } else {
      quickSiteDrafts.value.push(next);
    }
    activeQuickSiteId.value = next.id;
    structuredAvailable.value = true;
    lastEditMode.value = 'blocks';
    if (meta?.waf) {
      pendingWizardWaf.value = meta.waf.enabled ? { ...meta.waf } : null;
    }
    wizardLocalPreview.value = formatCaddyfile(generatedConfig.value);
    return next;
  }

  /**
   * 将多份发现草稿合并进工作台（纯本地，不 /load）。
   */
  function mergeDiscoveryDraftsIntoWorkbench(drafts: QuickSiteDraft[]) {
    for (const draft of drafts) {
      mergeWizardDraftIntoWorkbench(draft);
    }
    structuredAvailable.value = true;
    lastEditMode.value = 'blocks';
    mode.value = 'blocks';
    wizardLocalPreview.value = formatCaddyfile(generatedConfig.value);
  }

  /** 发现：仅写入会话草稿 */
  function handleDiscoveryCommitDrafts(drafts: QuickSiteDraft[]) {
    if (!drafts.length) return;
    mergeDiscoveryDraftsIntoWorkbench(drafts);
    message.success(`已写入 ${drafts.length} 个会话草稿（未调用 /load，未写 discovery DB）`);
    dockerDiscoveryOpen.value = false;
    onConfigReady?.();
  }

  /** 发现：dry-run Preview → 既有 previewBeforeSave（仅 /adapt） */
  async function handleDiscoveryPreview(drafts: QuickSiteDraft[]) {
    if (!drafts.length) return;
    mergeDiscoveryDraftsIntoWorkbench(drafts);
    dockerDiscoveryOpen.value = false;
    await previewBeforeSave('blocks');
  }

  /** 发现：用户确认后走既有 Apply_Path（Preview 确认弹窗，不直接 /load） */
  async function handleDiscoveryApply(drafts: QuickSiteDraft[]) {
    if (!drafts.length) return;
    mergeDiscoveryDraftsIntoWorkbench(drafts);
    dockerDiscoveryOpen.value = false;
    await previewBeforeSave('blocks');
  }

  /**
   * 向导仅写入草稿：合并进工作台后关闭向导，不调用 preview/push（无 /load）。
   */
  function handleWizardCommitDraft(
    draft: QuickSiteDraft,
    meta: { waf: SiteWizardWafDraft },
  ) {
    mergeWizardDraftIntoWorkbench(draft, meta);
    mode.value = 'blocks';
    message.success(
      pendingWizardWaf.value
        ? '站点草稿已写入工作台（含 WAF 意图，未热加载）'
        : '站点草稿已写入工作台（未调用 /load）',
    );
    siteWizardOpen.value = false;
    onConfigReady?.();
  }

  /**
   * 向导 Preview：合并草稿后走既有 previewBeforeSave（仅 /adapt，不 /load）。
   */
  async function handleWizardPreview(draft: QuickSiteDraft) {
    mergeWizardDraftIntoWorkbench(draft);
    mode.value = 'blocks';
    siteWizardOpen.value = false;
    await previewBeforeSave('blocks');
  }

  /**
   * 向导 Apply：合并草稿 → Preview 确认弹窗 → 用户确认后 confirmSave 才 /load。
   * 绝不在向导内直接 push。
   */
  async function handleWizardApply(
    draft: QuickSiteDraft,
    meta: { waf: SiteWizardWafDraft },
  ) {
    mergeWizardDraftIntoWorkbench(draft, meta);
    mode.value = 'blocks';
    siteWizardOpen.value = false;
    await previewBeforeSave('blocks');
    if (meta.waf?.enabled) {
      message.info(
        '站点已进入保存预览；WAF 意图请在「防火墙」页确认后单独应用，不会静默热加载。',
      );
    }
  }

  function duplicateQuickSite(site: QuickSiteDraft) {
    const clone = createQuickSiteDraft({
      ...site,
      domains: [...site.domains],
      id: genId(),
      name: `${site.name || '站点'}-copy`,
    });
    quickSiteDrafts.value.push(clone);
    activeQuickSiteId.value = clone.id;
  }

  function removeQuickSite(id: string) {
    const idx = quickSiteDrafts.value.findIndex((site) => site.id === id);
    if (idx < 0) return;
    quickSiteDrafts.value.splice(idx, 1);
    if (activeQuickSiteId.value === id) {
      activeQuickSiteId.value = quickSiteDrafts.value[0]?.id;
    }
  }

  function addUpstream() {
    formModel.value.upstreams.push({
      lbPolicy: 'round_robin',
      name: `upstream-${formModel.value.upstreams.length + 1}`,
      targets: ['localhost:8080'],
    });
  }

  function removeUpstream(index: number) {
    formModel.value.upstreams.splice(index, 1);
  }

  /** 上游池健康检查开关：启用时写入默认 path，关闭时清除 healthCheck */
  function isUpstreamHealthEnabled(index: number) {
    const path = formModel.value.upstreams[index]?.healthCheck?.path?.trim();
    return Boolean(path);
  }

  function setUpstreamHealthEnabled(index: number, enabled: boolean) {
    const upstream = formModel.value.upstreams[index];
    if (!upstream) return;
    if (enabled) {
      upstream.healthCheck = {
        path: upstream.healthCheck?.path?.trim() || defaultHealthCheck().path,
        interval: upstream.healthCheck?.interval || defaultHealthCheck().interval,
        timeout: upstream.healthCheck?.timeout || defaultHealthCheck().timeout,
      };
    } else {
      upstream.healthCheck = undefined;
    }
  }

  function ensureUpstreamHealth(index: number): HealthCheck {
    const upstream = formModel.value.upstreams[index];
    if (!upstream) return defaultHealthCheck();
    if (!upstream.healthCheck) {
      upstream.healthCheck = defaultHealthCheck();
    }
    return upstream.healthCheck;
  }

  /** Simple reverse_proxy 站点级（handle 级）健康检查开关 */
  function isSiteHealthEnabled(site: QuickSiteDraft) {
    return Boolean(site.healthCheck?.path?.trim());
  }

  function setSiteHealthEnabled(site: QuickSiteDraft, enabled: boolean) {
    if (enabled) {
      site.healthCheck = {
        path: site.healthCheck?.path?.trim() || defaultHealthCheck().path,
        interval: site.healthCheck?.interval || defaultHealthCheck().interval,
        timeout: site.healthCheck?.timeout || defaultHealthCheck().timeout,
      };
    } else {
      site.healthCheck = undefined;
    }
  }

  function ensureSiteHealth(site: QuickSiteDraft): HealthCheck {
    if (!site.healthCheck) {
      site.healthCheck = defaultHealthCheck();
    }
    return site.healthCheck;
  }

  function applyPreset() {
    const site = createQuickSiteDraft({
      id: genId(),
      domains: ['example.com'],
      mode: 'reverse_proxy',
      name: '默认站点',
      upstream: 'localhost:8080',
      lbPolicy: 'round_robin',
    });
    formModel.value = {
      schemaVersion: 1,
      global: { raw: '' },
      upstreams: [],
      sites: [],
    };
    preservedBlocks.value = [];
    quickSiteDrafts.value = [site];
    complexSiteSummaries.value = [];
    activeQuickSiteId.value = site.id;
    structuredAvailable.value = true;
    lastEditMode.value = 'blocks';
    mode.value = 'blocks';
  }

  function handleModeChange(value: unknown) {
    const next = value as CaddyPageMode;
    if (next === mode.value) return;
    if (next === 'waf') {
      mode.value = next;
      return;
    }
    if (next === 'raw') {
      lastEditMode.value = 'raw';
      mode.value = next;
      return;
    }
    if (next === 'blocks') {
      if (lastEditMode.value === 'raw') {
        if (!parseRawToBlocks(false)) return;
      } else {
        syncQuickStateFromForm(formModel.value);
      }
      lastEditMode.value = 'blocks';
      mode.value = next;
      return;
    }
    mode.value = next;
  }

  async function previewBeforeSave(kind: SavePreviewKind) {
    if (!selectedServerId.value) {
      message.warning('请先选择 Caddy 服务器');
      return;
    }

    const config = kind === 'raw' ? configContent.value : generatedConfig.value;
    const modules =
      kind === 'blocks'
        ? JSON.stringify({
            ...mergedQuickFormModel.value,
            preservedBlocks: preservedBlocks.value,
          })
        : '';

    if (kind === 'blocks') {
      const errors = quickValidationErrors.value;
      if (errors.length > 0) {
        message.error(`校验失败：${errors[0]}`);
        return;
      }
    }

    previewing.value = true;
    try {
      const preview = await previewCaddyConfigApi(selectedServerId.value, {
        config,
        mode: kind === 'blocks' ? 'quick' : 'raw',
        modules: modules || undefined,
      });
      if (preview?.valid === false) {
        message.error(`校验失败：${preview.errors?.[0] || 'Caddy 配置不可用'}`);
        return;
      }
      Object.assign(savePreview, {
        actions: preview?.actions ?? [],
        config: preview?.config || config,
        errors: preview?.errors ?? [],
        kind,
        modules,
        open: true,
        original: config,
      });
    } catch (error) {
      message.error(apiErrorMessage(error, '预览配置失败'));
    } finally {
      previewing.value = false;
    }
  }

  async function confirmSave() {
    if (!selectedServerId.value) return;
    saving.value = true;
    try {
      const content = savePreview.config || savePreview.original;
      const serverId = selectedServerId.value;
      await pushCaddyConfigApi(
        serverId,
        content,
        savePreview.kind === 'blocks' ? savePreview.modules : undefined,
      );
      message.success('配置已保存并自动热重载 Caddy');
      // 前缀匹配：config（含 simple-waf）、history、catalog、metrics
      await invalidateListDetailQueryKeys(queryClient, [
        qk.caddy.config(serverId),
        ['caddy', 'history', serverId],
        qk.caddy.catalog(serverId),
        ['caddy', 'metrics', serverId],
      ]);
      configContent.value = content;
      if (savePreview.kind === 'blocks' && savePreview.modules) {
        loadModules(savePreview.modules);
        lastEditMode.value = 'blocks';
        mode.value = 'blocks';
      } else {
        structuredAvailable.value = false;
        lastEditMode.value = 'raw';
        mode.value = 'raw';
      }
      savePreview.open = false;
    } catch (error) {
      message.error(apiErrorMessage(error, '保存配置失败'));
    } finally {
      saving.value = false;
    }
  }

  return {
    loadingConfig,
    configErrorMessage,
    saving,
    previewing,
    configContent,
    mode,
    lastEditMode,
    structuredAvailable,
    formModel,
    preservedBlocks,
    quickSiteDrafts,
    complexSiteSummaries,
    activeQuickSiteId,
    savePreview,
    siteWizardOpen,
    pendingWizardWaf,
    wizardLocalPreview,
    dockerDiscoveryOpen,
    dockerDiscoveryScanning,
    dockerDiscoveryResult,
    modeOptions,
    quickModeOptions,
    tlsModeOptions,
    lbPolicyOptions,
    upstreamPoolOptions,
    activeQuickSite,
    globalPreservedBlocks,
    preservedSiteBlocks,
    mergedQuickFormModel,
    hasPreservedContent,
    generatedConfig,
    previewConfig,
    quickValidationErrors,
    applyRouteModeDeepLink,
    syncQuickStateFromForm,
    applyDraft,
    parseRawToBlocks,
    loadModules,
    fetchConfig,
    addQuickSite,
    openSiteWizard,
    openDockerDiscovery,
    handleDockerDiscoveryScan,
    handleDiscoveryCommitDrafts,
    handleDiscoveryPreview,
    handleDiscoveryApply,
    handleWizardCommitDraft,
    handleWizardPreview,
    handleWizardApply,
    duplicateQuickSite,
    removeQuickSite,
    addUpstream,
    removeUpstream,
    isUpstreamHealthEnabled,
    setUpstreamHealthEnabled,
    ensureUpstreamHealth,
    isSiteHealthEnabled,
    setSiteHealthEnabled,
    ensureSiteHealth,
    applyPreset,
    handleModeChange,
    previewBeforeSave,
    confirmSave,
    blockKindLabel,
    blockKindColor,
    defaultHealthCheck,
  };
}
