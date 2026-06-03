<script setup lang="ts">
import { defineAsyncComponent, ref } from 'vue';
import { Icon } from '@iconify/vue';
import type { PreservedCaddyBlock } from '../types';

const VueMonacoEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoEditor)
);

const props = defineProps<{
  block: PreservedCaddyBlock;
}>();

const emit = defineEmits<{
  (e: 'update', id: string, raw: string): void;
}>();

const editing = ref(false);
const editDraft = ref('');

function startEdit() {
  editDraft.value = props.block.raw;
  editing.value = true;
}

function confirmEdit() {
  emit('update', props.block.id, editDraft.value);
  editing.value = false;
}

function cancelEdit() {
  editing.value = false;
}
</script>

<template>
  <div>
    <!-- 只读模式 -->
    <div v-if="!editing" class="rounded-md bg-gray-50 p-3">
      <div class="mb-2 flex justify-end">
        <Button size="tiny" text type="primary" @click="startEdit">
          <template #icon>
            <Icon icon="carbon:edit" class="text-xs" />
          </template>
          编辑
        </Button>
      </div>
      <pre style="white-space:pre-wrap;word-break:break-all;max-height:400px;overflow:auto;background:#f5f5f5;padding:12px;border-radius:6px;font-size:12px;">{{ block.raw }}</pre>
    </div>

    <!-- 编辑模式 -->
    <div v-else>
      <div class="relative h-[200px] rounded-md border border-primary-200">
        <VueMonacoEditor
          v-model:value="editDraft"
          language="shell"
          theme="vs"
          :options="{
            automaticLayout: true,
            fixedOverflowWidgets: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            wordWrap: 'on',
            fontSize: 13
          }"
          class="absolute inset-0"
        />
      </div>
      <div class="mt-2 flex justify-end gap-2">
        <Button size="tiny" @click="cancelEdit">取消</Button>
        <Button size="tiny" type="primary" @click="confirmEdit">应用</Button>
      </div>
    </div>
  </div>
</template>
