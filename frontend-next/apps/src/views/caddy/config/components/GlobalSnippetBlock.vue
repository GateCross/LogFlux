<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue';
import { Icon } from '@iconify/vue';
import PreservedBlockEditor from './PreservedBlockEditor.vue';
import type { PreservedCaddyBlock } from '../types';

const VueMonacoEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoEditor)
);

const props = defineProps<{
  globalRaw?: string;
  preservedBlocks: PreservedCaddyBlock[];
  readOnly?: boolean;
  initialGlobalRaw?: string;
}>();

const emit = defineEmits<{
  (e: 'compare'): void;
  (e: 'restore'): void;
  (e: 'update:globalRaw', value: string): void;
  (e: 'updatePreservedBlock', id: string, raw: string): void;
}>();

const globalRawModel = computed<string>({
  get: () => props.globalRaw ?? '',
  set: (val) => emit('update:globalRaw', val)
});

const globalChanged = computed(() =>
  (globalRawModel.value ?? '').trim() !== (props.initialGlobalRaw ?? '').trim()
);

const preservedNonSite = computed(() =>
  props.preservedBlocks.filter(b => b.kind !== 'site')
);

/** global.raw 是否只有注释和空行（无实际配置指令） */
const isOnlyComments = computed(() => {
  const text = globalRawModel.value ?? '';
  return text.split('\n').every(line => {
    const t = line.trim();
    return !t || t.startsWith('#');
  });
});
</script>

<template>
  <Card size="small" :bordered="false">
    <template #header>
      <div class="flex items-center justify-between gap-3">
        <span class="font-semibold">全局配置</span>
        <div class="flex items-center gap-2">
          <Tag v-if="globalChanged && !readOnly" type="warning" size="small" :bordered="false">未保存</Tag>
          <Button v-if="!readOnly" size="tiny" secondary :disabled="!initialGlobalRaw" @click="emit('restore')">
            恢复已保存
          </Button>
          <Button v-if="!readOnly" size="tiny" :disabled="!globalRawModel && !initialGlobalRaw" @click="emit('compare')">
            对比
          </Button>
        </div>
      </div>
    </template>

    <Alert v-if="readOnly" type="info" :show-icon="true" class="mb-3">
      全局配置仅在"快速配置"模式下可编辑；原始配置模式请直接维护完整 Caddyfile。
    </Alert>

    <Alert v-if="isOnlyComments && preservedNonSite.length === 0" type="info" :show-icon="true" class="mb-3">
      当前无全局配置指令。所有全局选项和片段已在站点块中以 import 方式管理。
    </Alert>

    <div v-if="!isOnlyComments" class="relative h-[240px]">
      <VueMonacoEditor
        v-model:value="globalRawModel"
        language="shell"
        theme="vs"
        :options="{
          automaticLayout: true,
          fixedOverflowWidgets: true,
          readOnly: readOnly ?? false,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          wordWrap: 'on'
        }"
        class="absolute inset-0"
      />
    </div>

    <!-- 只读保留块（snippet / 全局 / 未知） -->
    <div v-if="preservedNonSite.length" class="mt-4">
      <div class="mb-3 flex items-center gap-2">
        <span class="text-xs text-gray-500 font-semibold">只读保留块</span>
        <Tag size="tiny" type="warning" :bordered="false" round>{{ preservedNonSite.length }}</Tag>
      </div>
      <Collapse>
        <Collapse.Panel
          v-for="block in preservedNonSite"
          :key="block.id"
          :name="block.id"
        >
          <template #header>
            <div class="flex items-center gap-2">
              <Tag
                size="tiny"
                :type="block.kind === 'snippet' ? 'info' : block.kind === 'global' ? 'success' : 'warning'"
                :bordered="false"
                round
              >
                {{ block.kind === 'snippet' ? 'snippet' : block.kind === 'global' ? '全局' : '未知' }}
              </Tag>
              <span class="text-sm font-medium">{{ block.title }}</span>
            </div>
          </template>
          <template #header-extra>
            <Tooltip trigger="hover">
              <template #trigger>
                <Icon icon="carbon:information" class="text-gray-400" />
              </template>
              {{ block.reason }}
            </Tooltip>
          </template>
          <PreservedBlockEditor :block="block" @update="(id, raw) => emit('updatePreservedBlock', id, raw)" />
        </Collapse.Panel>
      </Collapse>
    </div>
  </Card>
</template>
