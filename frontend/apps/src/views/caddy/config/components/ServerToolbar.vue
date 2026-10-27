<script lang="ts" setup>
import { computed } from 'vue';

import { Button } from 'antdv-next';

const props = defineProps<{
  configContent: string;
  mode: string;
  previewing: boolean;
}>();

const emit = defineEmits<{
  applyPreset: [];
  openHistory: [];
  parseRaw: [];
  saveBlocks: [];
  saveRaw: [];
}>();

/** 保存按钮文案随编辑视角变化；防火墙有独立保存动作 */
const saveLabel = computed(() =>
  props.mode === 'raw' ? '保存原始配置' : '保存分块配置',
);
</script>

<template>
  <div class="caddy-toolbar">
    <div class="toolbar-group actions">
      <Button @click="emit('openHistory')">历史</Button>
      <Button v-if="props.mode === 'blocks'" @click="emit('applyPreset')">
        默认模板
      </Button>
      <Button
        v-if="props.mode === 'raw'"
        :disabled="!props.configContent.trim()"
        @click="emit('parseRaw')"
      >
        解析为分块配置
      </Button>
      <!-- 防火墙有自己的保存动作，这里不显示 Caddyfile 保存按钮 -->
      <Button
        v-if="props.mode !== 'waf'"
        type="primary"
        :loading="props.previewing"
        @click="props.mode === 'raw' ? emit('saveRaw') : emit('saveBlocks')"
      >
        {{ saveLabel }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.caddy-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  flex-wrap: wrap;
  padding-bottom: 16px;
  margin-bottom: 16px;
  border-bottom: 1px solid hsl(var(--border));
}

.toolbar-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

@media (max-width: 960px) {
  .toolbar-group.actions {
    justify-content: flex-start;
  }
}
</style>
