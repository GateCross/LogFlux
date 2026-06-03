<script setup lang="ts">
import { defineAsyncComponent } from 'vue';

const VueMonacoEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoEditor)
);

const props = defineProps<{
  configContent: string;
}>();
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="relative min-h-0 flex-1">
      <VueMonacoEditor
        :value="props.configContent"
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
  </div>
</template>
