<script setup lang="ts">
import { defineAsyncComponent } from 'vue';
import type { useCaddyConfigHistory } from '../composables/useCaddyConfigHistory';

const VueMonacoEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoEditor)
);
const VueMonacoDiffEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoDiffEditor)
);

/** 接受 composable 返回值作为单个 prop，避免逐字段透传 */
defineProps<{
  history: ReturnType<typeof useCaddyConfigHistory>;
  historyCompareRight: string;
}>();
</script>

<template>
  <div>
    <!-- 历史列表 -->
    <Modal
      v-model:open="history.showHistoryModal.value"
      title="配置历史"
      width="90vw"
      :body-style="{ maxWidth: '48rem', margin: '0 auto' }"
    >
      <Table
        :columns="history.historyColumns"
        :data-source="history.historyList.value"
        :loading="history.historyLoading.value"
        :pagination="{
          current: history.historyPagination.value.page,
          pageSize: history.historyPagination.value.pageSize,
          total: history.historyPagination.value.itemCount,
          onChange: (page: number) => history.handleHistoryPageChange(page)
        }"
        size="small"
      />
    </Modal>

    <!-- 历史详情 -->
    <Modal
      v-model:open="history.showHistoryDetailModal.value"
      :title="history.historyDetail.value ? `历史配置预览 - ${history.historyDetail.value.createdAt}` : '历史配置预览'"
      width="90vw"
      :body-style="{ maxWidth: '64rem', margin: '0 auto' }"
    >
      <div class="mb-3 flex flex-wrap items-center gap-3 text-xs text-gray-500">
        <span>动作：{{ history.historyDetail.value ? history.formatHistoryAction(history.historyDetail.value.action) : '-' }}</span>
        <span>时间：{{ history.historyDetail.value?.createdAt ?? '-' }}</span>
      </div>
      <div class="relative h-[60vh]">
        <VueMonacoEditor
          :value="history.historyDetailFormattedConfig.value"
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
    </Modal>

    <!-- 历史对比 -->
    <Modal
      v-model:open="history.showHistoryCompareModal.value"
      title="历史配置对比"
      width="90vw"
      :body-style="{ maxWidth: '64rem', margin: '0 auto' }"
    >
      <div class="diff-head">
        <div>历史版本</div>
        <div class="flex items-center justify-between">
          <span>当前配置</span>
          <Switch :value="history.historyDiffOnly.value" size="small" @update:value="history.historyDiffOnly.value = $event">
            <template #checked>仅差异</template>
            <template #unchecked>全部</template>
          </Switch>
        </div>
      </div>
      <div class="relative h-[65vh]">
        <VueMonacoDiffEditor
          :original="history.historyCompareLeftFormatted.value"
          :modified="historyCompareRight"
          language="shell"
          theme="vs"
          :options="{
            automaticLayout: true,
            readOnly: true,
            renderSideBySide: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            wordWrap: 'on',
            hideUnchangedRegions: { enabled: history.historyDiffOnly.value }
          }"
          class="absolute inset-0"
        />
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.diff-head {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  font-size: 12px;
  color: #64748b;
  margin-bottom: 8px;
}

/* NModal teleport 到 body 后脱离父组件 DOM 子树，需在此补充 Monaco 浮层层级 */
:deep(.monaco-editor-overlay) {
  z-index: 1000 !important;
}
</style>
