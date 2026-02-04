<script setup lang="ts">
import { VueMonacoEditor } from '@guolao/vue-monaco-editor';

const content = defineModel<string>({ required: true });
const props = defineProps<{ readOnly?: boolean }>();
</script>

<template>
  <div class="h-full relative">
    <VueMonacoEditor
      v-model:value="content"
      language="shell"
      theme="vs"
      :options="{
        automaticLayout: true,
        fixedOverflowWidgets: true,
        readOnly: props.readOnly ?? false,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        wordWrap: 'on'
      }"
      class="absolute inset-0"
    />
  </div>
</template>
