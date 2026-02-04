<script setup lang="ts">
import { VueMonacoEditor } from '@guolao/vue-monaco-editor';

const props = defineProps<{
  configContent: string;
}>();
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="relative flex-1 min-h-0">
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
