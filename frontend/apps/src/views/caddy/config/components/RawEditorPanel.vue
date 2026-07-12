<script lang="ts" setup>
import MonacoCodeEditor from '#/components/code-editor/MonacoCodeEditor.vue';

const props = defineProps<{
  modelValue: string;
  /** preview 模式只读展示 */
  readonly?: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

function onUpdateValue(value: string) {
  emit('update:modelValue', value ?? '');
}
</script>

<template>
  <div :class="props.readonly ? 'preview-pane' : 'raw-pane'">
    <MonacoCodeEditor
      :model-value="props.modelValue"
      :readonly="props.readonly"
      language="plaintext"
      :height="props.readonly ? 520 : 560"
      @update:model-value="onUpdateValue"
    />
  </div>
</template>

<style scoped>
.raw-pane,
.preview-pane {
  width: 100%;
}
</style>
