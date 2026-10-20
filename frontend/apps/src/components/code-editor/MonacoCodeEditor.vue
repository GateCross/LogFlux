<script lang="ts" setup>
/**
 * Monaco 代码编辑器封装；加载失败回退 Textarea
 */
import { computed, ref } from 'vue';

import { usePreferences } from '@vben/preferences';

import { Alert, Input } from 'antdv-next';
import { VueMonacoEditor } from '@guolao/vue-monaco-editor';

import '#/components/code-editor/monaco-local';

const props = withDefaults(
  defineProps<{
    modelValue: string;
    readonly?: boolean;
    /** monaco language id；默认 plaintext */
    language?: string;
    height?: string | number;
  }>(),
  {
    readonly: false,
    language: 'plaintext',
    height: 480,
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const editorAttempt = ref(0);
const loadErrorMessage = '代码编辑器加载失败，已切换为纯文本编辑。';

/** 跟随应用主题，避免深色模式下编辑器仍是白底 */
const { isDark } = usePreferences();
const editorTheme = computed(() => (isDark.value ? 'vs-dark' : 'vs'));

const editorHeight = computed(() => {
  const h = props.height;
  if (typeof h === 'number') return `${h}px`;
  return h || '480px';
});

const monacoOptions = computed(() => ({
  automaticLayout: true,
  readOnly: props.readonly,
  minimap: { enabled: false },
  scrollBeyondLastLine: false,
  wordWrap: 'on' as const,
  fontSize: 13,
  fontFamily:
    "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
  // 只读时仍允许滚动与选中复制
  domReadOnly: props.readonly,
}));

function onUpdateValue(value: string | undefined) {
  emit('update:modelValue', value ?? '');
}

function onFallbackInput(value: string | number | undefined) {
  emit('update:modelValue', value == null ? '' : String(value));
}

function retryEditorLoad() {
  // failure slot 中递增 key，让编辑器重新挂载并重新初始化
  editorAttempt.value += 1;
}

/** 供单测 / 调用方：无编辑时读出应与 model 等价（Property 3 语义） */
function getFullText(): string {
  return props.modelValue ?? '';
}

defineExpose({ getFullText });
</script>

<template>
  <div class="monaco-code-editor" :style="{ minHeight: editorHeight }">
    <VueMonacoEditor
      :key="editorAttempt"
      :value="props.modelValue"
      :language="props.language"
      :theme="editorTheme"
      :height="editorHeight"
      :width="'100%'"
      :options="monacoOptions"
      class-name="monaco-code-editor__inner"
      @update:value="onUpdateValue"
    >
      <template #default>
        <div class="monaco-code-editor__loading">正在加载代码编辑器…</div>
      </template>
      <template #failure>
        <div class="monaco-code-editor__failure">
          <Alert
            type="warning"
            show-icon
            class="mb-2"
            :message="loadErrorMessage"
          />
          <Input.TextArea
            :value="props.modelValue"
            :readonly="props.readonly"
            :auto-size="{ minRows: 16, maxRows: 40 }"
            class="code-textarea"
            :placeholder="props.readonly ? undefined : '请输入内容'"
            @update:value="onFallbackInput"
          />
          <button
            type="button"
            class="monaco-code-editor__retry"
            @click="retryEditorLoad"
          >
            重试加载编辑器
          </button>
        </div>
      </template>
    </VueMonacoEditor>
  </div>
</template>

<style scoped>
.monaco-code-editor {
  width: 100%;
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  overflow: hidden;
  background: hsl(var(--card));
}

.monaco-code-editor__inner {
  min-height: inherit;
}

.monaco-code-editor__loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  color: hsl(var(--muted-foreground));
  font-size: 13px;
}

.monaco-code-editor__failure {
  padding: 12px;
}

.mb-2 {
  margin-bottom: 8px;
}

.code-textarea {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono',
    'Courier New', monospace;
  font-size: 13px;
  width: 100%;
}

.monaco-code-editor__retry {
  margin-top: 8px;
  padding: 4px 12px;
  font-size: 13px;
  color: hsl(var(--primary));
  background: transparent;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  cursor: pointer;
}

.monaco-code-editor__retry:hover {
  background: hsl(var(--accent));
}
</style>
