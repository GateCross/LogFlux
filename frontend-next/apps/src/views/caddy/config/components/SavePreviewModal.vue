<script setup lang="ts">
import { defineAsyncComponent } from 'vue';
import type { SavePreviewState } from '../composables/useCaddyPublishFlow';

const VueMonacoEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoEditor)
);

const props = defineProps<{
  saving: boolean;
}>();

const savePreview = defineModel<SavePreviewState>('savePreview', { required: true });

const emit = defineEmits<{
  (e: 'confirm'): void;
  (e: 'close'): void;
}>();
</script>

<template>
  <Modal v-model:show="savePreview.visible" preset="card" title="保存预览" class="max-w-5xl w-[90vw]">
    <div class="mb-3 flex flex-wrap items-center gap-2 text-xs text-gray-500">
      <Tag size="small" type="info" :bordered="false">
        {{ savePreview.kind === 'blocks' ? '快速配置' : '原始配置' }}
      </Tag>
      <span v-if="savePreview.actions.length">动作：{{ savePreview.actions.join(' / ') }}</span>
    </div>
    <Alert v-if="savePreview.errors.length" type="error" :show-icon="true" class="mb-3">
      {{ savePreview.errors[0] }}
    </Alert>
    <div class="relative h-[60vh]">
      <VueMonacoEditor
        :value="savePreview.config"
        language="shell"
        theme="vs"
        :options="{
          automaticLayout: true,
          fixedOverflowWidgets: true,
          readOnly: true,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          wordWrap: 'on'
        }"
        class="absolute inset-0"
      />
    </div>
    <div class="mt-4 flex justify-end gap-2">
      <Button secondary :disabled="props.saving" @click="emit('close')">取消</Button>
      <Button type="primary" :loading="props.saving" @click="emit('confirm')">确认保存</Button>
    </div>
  </Modal>
</template>

<style scoped>
:deep(.monaco-editor-overlay) {
  z-index: 1000 !important;
}
</style>
