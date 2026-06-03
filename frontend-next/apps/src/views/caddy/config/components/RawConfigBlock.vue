<script setup lang="ts">
import { defineAsyncComponent } from 'vue';

const VueMonacoEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoEditor)
);

const content = defineModel<string>({ required: true });

const emit = defineEmits<{
  (e: 'reparse'): void;
}>();
</script>

<template>
  <div class="h-full flex flex-col gap-2">
    <div class="flex items-center justify-between">
      <div class="text-xs text-gray-500">直接维护完整 Caddyfile，适合高级规则。</div>
      <Button size="small" @click="emit('reparse')">从原始配置解析为分块</Button>
    </div>
    <div class="relative min-h-0 flex-1">
      <VueMonacoEditor
        v-model:value="content"
        language="shell"
        theme="vs"
        :options="{
          automaticLayout: true,
          fixedOverflowWidgets: true,
          readOnly: false,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          wordWrap: 'on'
        }"
        class="absolute inset-0"
      />
    </div>
  </div>
</template>
