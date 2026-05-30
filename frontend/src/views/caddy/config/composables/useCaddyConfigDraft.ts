import { computed, ref, type Ref } from 'vue';
import { useMessage } from 'naive-ui';
import { fetchCaddyConfig } from '@/service/api/caddy';
import type { CaddyBlockDraft, CaddyFormModel, CaddyPageMode } from '../types';
import {
  type DiffRow,
  buildCaddyfile,
  buildLineDiff,
  formatCaddyfile,
  normalizeModules,
  validateStructuredConfig
} from '../caddy-config-utils';
import {
  type ComplexSiteSummary,
  type QuickSiteDraft,
  buildQuickConfigState,
  createQuickSiteDraft,
  mergeQuickConfigDrafts
} from '../quick-config-utils';
import { parseCaddyfileToBlocks } from '../caddy-config-blocks';

const createEmptyFormModel = (): CaddyFormModel => ({
  schemaVersion: 1,
  global: { raw: '' },
  upstreams: [],
  sites: []
});

export function useCaddyConfigDraft(currentServerId: Ref<number | null>) {
  const message = useMessage();
  const loading = ref(false);
  const configContent = ref('');
  const mode = ref<CaddyPageMode>('blocks');
  const lastEditMode = ref<'blocks' | 'raw'>('blocks');
  const structuredAvailable = ref(false);

  const formModel = ref<CaddyFormModel>(createEmptyFormModel());
  const preservedBlocks = ref<CaddyBlockDraft['preservedBlocks']>([]);
  const quickSiteDrafts = ref<QuickSiteDraft[]>([]);
  const complexSiteSummaries = ref<ComplexSiteSummary[]>([]);
  const activeQuickSiteId = ref<string | null>(null);
  const initialGlobalRaw = ref('');

  const structuredReady = computed(() => {
    if (structuredAvailable.value) return true;
    const model = formModel.value;
    if (model.sites?.length) return true;
    if (model.upstreams?.length) return true;
    return Boolean(model.global?.raw?.trim());
  });

  const mergedQuickFormModel = computed(() => mergeQuickConfigDrafts(formModel.value, quickSiteDrafts.value));
  const generatedQuickCaddyfile = computed(() => buildCaddyfile(mergedQuickFormModel.value));
  const effectiveConfigContent = computed(() =>
    lastEditMode.value === 'raw' ? configContent.value : generatedQuickCaddyfile.value
  );
  const formattedConfigContent = computed(() => formatCaddyfile(effectiveConfigContent.value));

  const globalDiffRows = computed<DiffRow[]>(() => {
    const rows = buildLineDiff(initialGlobalRaw.value ?? '', formModel.value.global?.raw ?? '');
    return rows;
  });

  const quickValidationErrors = computed(() => {
    if (!structuredReady.value && configContent.value.trim()) {
      return [];
    }
    return validateStructuredConfig(mergedQuickFormModel.value);
  });

  function syncQuickStateFromForm(model: CaddyFormModel) {
    const { simpleSites, complexSites } = buildQuickConfigState(model);
    const nextActiveId = simpleSites.some(item => item.id === activeQuickSiteId.value)
      ? activeQuickSiteId.value
      : simpleSites[0]?.id || null;

    quickSiteDrafts.value = simpleSites;
    complexSiteSummaries.value = complexSites;
    activeQuickSiteId.value = nextActiveId;
  }

  async function loadConfig() {
    if (!currentServerId.value) return;

    loading.value = true;
    const { data, error } = await fetchCaddyConfig(currentServerId.value);
    loading.value = false;

    if (error) {
      message.error('获取配置失败');
      return;
    }
    if (!data) return;

    configContent.value = data.config || '';
    structuredAvailable.value = false;
    formModel.value = createEmptyFormModel();
    preservedBlocks.value = [];
    activeQuickSiteId.value = null;

    if (data.modules) {
      try {
        const parsed = JSON.parse(data.modules);
        if (parsed?.sites || parsed?.global) {
          const normalized = normalizeModules(parsed);
          formModel.value = normalized;
          structuredAvailable.value = true;
        }
      } catch {
        message.warning('结构化配置解析失败，已忽略');
        formModel.value = createEmptyFormModel();
        structuredAvailable.value = false;
      }
    }

    initialGlobalRaw.value = formModel.value.global?.raw ?? '';

    // 始终从原始配置中解析 preservedBlocks（复杂站点、snippet 等只读块）
    if (configContent.value.trim()) {
      const parsed = parseCaddyfileToBlocks(configContent.value);
      const allPreserved = parsed.preservedBlocks ?? [];

      if (structuredAvailable.value) {
        // modules 已有 formModel.sites，过滤掉与之重复的 preserved site 块
        const siteDomains = new Set(formModel.value.sites.flatMap(s => s.domains));
        preservedBlocks.value = allPreserved.filter(b => {
          if (b.kind !== 'site') return true;
          // preservedBlock title 的第一个 token 是域名
          const firstDomain = b.title.split(/[\s,]+/)[0];
          return !siteDomains.has(firstDomain);
        });
      } else {
        preservedBlocks.value = allPreserved;
      }

      // 若 modules 路径未设置 sites，则用解析结果补充
      if (!structuredAvailable.value && (parsed.sites.length > 0 || parsed.global?.raw)) {
        applyDraft(parsed, false);
        return;
      }
    }

    syncQuickStateFromForm(formModel.value);

    // configContent 保持 data.config 原样，不重建，不改顺序
    mode.value = 'blocks';
    lastEditMode.value = 'blocks';
  }

  /** 从原始配置解析为分块草稿 */
  function ensureBlocksFromRaw(force = false) {
    if (!force && structuredReady.value && formModel.value.sites.length > 0) return;
    if (!configContent.value.trim()) return;
    const draft = parseCaddyfileToBlocks(configContent.value);
    if (draft.sites.length === 0 && !draft.global?.raw && draft.preservedBlocks.length === 0) return;
    applyDraft(draft, false);
  }

  function applyDraft(draft: CaddyBlockDraft, notify?: boolean) {
    formModel.value = {
      schemaVersion: draft.schemaVersion,
      global: draft.global,
      upstreams: draft.upstreams,
      sites: draft.sites
    };
    preservedBlocks.value = draft.preservedBlocks ?? [];
    structuredAvailable.value = true;
    initialGlobalRaw.value = draft.global?.raw ?? '';
    syncQuickStateFromForm(formModel.value);
    // 不覆盖 configContent：它是服务端源文件内容，原始配置和预览页应原样显示
    lastEditMode.value = 'blocks';
    mode.value = 'blocks';
    if (notify) message.success('已从原始配置解析');
  }

  function addQuickSite() {
    const draft = createQuickSiteDraft({
      id: genIdSafe(),
      name: `新站点-${quickSiteDrafts.value.length + 1}`,
      domains: [],
      mode: 'reverse_proxy',
      upstream: 'localhost:8080'
    });
    quickSiteDrafts.value.push(draft);
    activeQuickSiteId.value = draft.id;
    lastEditMode.value = 'blocks';
    mode.value = 'blocks';
  }

  function duplicateQuickSite(id: string) {
    const target = quickSiteDrafts.value.find(item => item.id === id);
    if (!target) return;
    const clone = createQuickSiteDraft({
      ...target,
      domains: [...target.domains],
      id: genIdSafe(),
      name: `${target.name || '站点'}-copy`
    });
    quickSiteDrafts.value.push(clone);
    activeQuickSiteId.value = clone.id;
  }

  function removeQuickSite(id: string) {
    const idx = quickSiteDrafts.value.findIndex(item => item.id === id);
    if (idx < 0) return;
    quickSiteDrafts.value.splice(idx, 1);
    if (activeQuickSiteId.value === id) {
      activeQuickSiteId.value = quickSiteDrafts.value[0]?.id || null;
    }
  }

  function resetState() {
    configContent.value = '';
    formModel.value = createEmptyFormModel();
    preservedBlocks.value = [];
    syncQuickStateFromForm(formModel.value);
  }

  /** 更新 preservedBlock 的 raw 内容（编辑保留块后回调） */
  function updatePreservedBlock(id: string, raw: string) {
    const idx = preservedBlocks.value.findIndex(b => b.id === id);
    if (idx < 0) return;
    preservedBlocks.value[idx] = { ...preservedBlocks.value[idx], raw };
  }

  return {
    loading,
    configContent,
    mode,
    lastEditMode,
    structuredAvailable,
    formModel,
    preservedBlocks,
    quickSiteDrafts,
    complexSiteSummaries,
    activeQuickSiteId,
    initialGlobalRaw,
    structuredReady,
    mergedQuickFormModel,
    generatedQuickCaddyfile,
    effectiveConfigContent,
    formattedConfigContent,
    globalDiffRows,
    quickValidationErrors,
    syncQuickStateFromForm,
    loadConfig,
    ensureBlocksFromRaw,
    applyDraft,
    addQuickSite,
    duplicateQuickSite,
    removeQuickSite,
    resetState,
    updatePreservedBlock
  };
}

function genIdSafe(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}
