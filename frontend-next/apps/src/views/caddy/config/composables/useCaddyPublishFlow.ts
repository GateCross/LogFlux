import { ref, type Ref } from 'vue';
import { Modal, message } from 'ant-design-vue';
import { previewCaddyConfigApi, pushCaddyConfigApi, pushCaddyConfigApi } from '#/api/caddy/server';
import type { CaddyBlockDraft, CaddyFormModel, CaddyPageMode } from '../types';
import { normalizeModules, validateStructuredConfig } from '../caddy-config-utils';
import { buildCaddyfileFromBlocks } from '../caddy-config-blocks';

export interface SavePreviewState {
  visible: boolean;
  kind: 'blocks' | 'raw';
  config: string;
  actions: string[];
  errors: string[];
  modules: string;
  original: string;
}

export function useCaddyPublishFlow(opts: {
  currentServerId: Ref<number | null>;
  configContent: Ref<string>;
  mode: Ref<CaddyPageMode>;
  lastEditMode: Ref<'blocks' | 'raw'>;
  structuredAvailable: Ref<boolean>;
  formModel: Ref<CaddyFormModel>;
  preservedBlocks: Ref<CaddyBlockDraft['preservedBlocks']>;
  initialGlobalRaw: Ref<string>;
  mergedQuickFormModel: Ref<CaddyFormModel>;
  structuredReady: Ref<boolean>;
  syncQuickStateFromForm: (model: CaddyFormModel) => void;
  ensureBlocksFromRaw: (force?: boolean) => void;
}) {
  
  
  const saving = ref(false);

  const savePreview = ref<SavePreviewState>({
    visible: false,
    kind: 'blocks',
    config: '',
    actions: [],
    errors: [],
    modules: '',
    original: ''
  });

  function confirmOverwriteStructured(actionLabel: string, onConfirm: () => void) {
    if (!opts.structuredReady.value) {
      onConfirm();
      return;
    }
    dialog.warning({
      title: '覆盖确认',
      content: `${actionLabel}将覆盖当前结构化配置，未保存内容会丢失，是否继续？`,
      positiveText: '继续',
      negativeText: '取消',
      onPositiveClick: onConfirm
    });
  }

  /** 分块模式保存 */
  async function saveBlocksConfig() {
    if (!opts.currentServerId.value) {
      message.warning('请先选择 Caddy 服务器');
      return;
    }

    const nextFormModel = opts.mergedQuickFormModel.value;
    const hasPreservedContent = opts.preservedBlocks.value.some(b => b.raw.trim().length > 0);
    const errors = validateStructuredConfig(nextFormModel, hasPreservedContent);
    if (errors.length > 0) {
      message.error(`校验失败：${errors[0]}`);
      return;
    }

    // 构建完整 Caddyfile（含 preserved blocks），按源文件顺序排版以避免打乱用户原有顺序
    const draftForBuild: CaddyBlockDraft = {
      ...nextFormModel,
      preservedBlocks: opts.preservedBlocks.value
    };
    const content = buildCaddyfileFromBlocks(draftForBuild, { sourceOrder: opts.configContent.value });
    const hasRealContent = content.split('\n').some(line => {
      const t = line.trim();
      return t && !t.startsWith('#');
    });
    if (!hasRealContent) {
      message.warning('当前无有效配置内容，请先添加站点或编辑全局配置');
      return;
    }

    saving.value = true;
    // modules 包含完整分块数据：可编辑表单 + 只读保留块
    const modules = JSON.stringify({
      ...nextFormModel,
      preservedBlocks: opts.preservedBlocks.value
    });
    const { data: preview, error: previewError } = await previewCaddyConfigApi(opts.currentServerId.value, {
      mode: 'quick',
      config: content,
      modules
    });
    if (previewError || !preview) {
      saving.value = false;
      message.error('预览配置失败');
      return;
    }
    if (!preview.valid) {
      saving.value = false;
      message.error(`校验失败：${preview.errors?.[0] || 'Caddy 配置不可用'}`);
      return;
    }

    savePreview.value = {
      visible: true,
      kind: 'blocks',
      config: preview.config || content,
      actions: preview.actions || [],
      errors: preview.errors || [],
      original: content,
      modules
    };
    saving.value = false;
  }

  /** 原始配置模式保存 */
  async function saveRawConfig() {
    if (!opts.currentServerId.value) return;

    saving.value = true;
    const { data: preview, error: previewError } = await previewCaddyConfigApi(opts.currentServerId.value, {
      mode: 'raw',
      config: opts.configContent.value
    });
    if (previewError || !preview) {
      saving.value = false;
      message.error('预览配置失败');
      return;
    }
    if (!preview.valid) {
      saving.value = false;
      message.error(`校验失败：${preview.errors?.[0] || 'Caddy 配置不可用'}`);
      return;
    }

    savePreview.value = {
      visible: true,
      kind: 'raw',
      config: preview.config || opts.configContent.value,
      actions: preview.actions || [],
      errors: preview.errors || [],
      original: opts.configContent.value,
      modules: ''
    };
    saving.value = false;
  }

  /** 确认保存 */
  async function confirmSavePreview(): Promise<boolean> {
    if (!opts.currentServerId.value) return false;

    saving.value = true;
    let saved = false;
    try {
      if (savePreview.value.kind === 'raw') {
        const { error } = await pushCaddyConfigApi(
          opts.currentServerId.value,
          savePreview.value.config || savePreview.value.original
        );
        if (error) {
          message.error('保存配置失败');
          return false;
        }
        message.success('配置已保存并自动热重载 Caddy');
        opts.configContent.value = savePreview.value.config || savePreview.value.original;
        opts.structuredAvailable.value = false;
        opts.lastEditMode.value = 'raw';
        opts.mode.value = 'raw';
        saved = true;
        return true;
      }

      const { error } = await pushCaddyConfigApi(
        opts.currentServerId.value,
        savePreview.value.config || savePreview.value.original,
        savePreview.value.modules
      );
      if (error) {
        message.error('保存配置失败');
        return false;
      }

      message.success('配置已保存并自动热重载 Caddy');
      if (savePreview.value.modules) {
        try {
          opts.formModel.value = normalizeModules(JSON.parse(savePreview.value.modules));
        } catch {
          // 保持原表单状态
        }
      }
      opts.configContent.value = savePreview.value.config || savePreview.value.original;
      opts.structuredAvailable.value = true;
      opts.initialGlobalRaw.value = opts.formModel.value.global?.raw ?? '';
      opts.syncQuickStateFromForm(opts.formModel.value);
      opts.lastEditMode.value = 'blocks';
      opts.mode.value = 'blocks';
      saved = true;
      return true;
    } finally {
      saving.value = false;
      if (saved) {
        savePreview.value.visible = false;
      }
    }
  }

  function closeSavePreview() {
    if (saving.value) return;
    savePreview.value.visible = false;
  }

  function handleModeChange(nextMode: 'blocks' | 'waf' | 'raw' | 'preview') {
    if (nextMode === opts.mode.value) return;

    if (nextMode === 'waf') {
      opts.mode.value = 'waf';
      return;
    }

    if (nextMode === 'raw') {
      // 保留 configContent 当前内容（来自服务端原始配置或用户编辑），
      // 不重建以避免打乱原始 Caddyfile 的块顺序
      opts.lastEditMode.value = 'raw';
      opts.mode.value = 'raw';
      return;
    }

    if (nextMode === 'blocks') {
      if (opts.lastEditMode.value === 'raw') {
        opts.ensureBlocksFromRaw(true);
      } else {
        opts.syncQuickStateFromForm(opts.formModel.value);
      }
      opts.lastEditMode.value = 'blocks';
      opts.mode.value = 'blocks';
      return;
    }

    opts.mode.value = 'preview';
  }

  return {
    saving,
    savePreview,
    confirmOverwriteStructured,
    saveBlocksConfig,
    saveRawConfig,
    confirmSavePreview,
    closeSavePreview,
    handleModeChange
  };
}
