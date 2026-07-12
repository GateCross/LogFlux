<script lang="ts" setup>
import { RadioGroup, RadioButton } from 'antdv-next';

import type { CaddyPageMode } from '../types';

const props = defineProps<{
  mode: CaddyPageMode;
  modeOptions: Array<{ label: string; value: string }>;
}>();

const emit = defineEmits<{
  change: [value: CaddyPageMode];
}>();

const summaries: Record<CaddyPageMode, string> = {
  blocks: '编辑常用站点能力，复杂配置自动保留。',
  waf: '配置 Coraza / OWASP CRS 常用开关并应用到 Caddy。',
  raw: '直接维护完整 Caddyfile。',
  preview: '查看当前将要发布的 Caddyfile。',
};

function onModeChange(event: any) {
  const next = event?.target?.value ?? event;
  emit('change', next as CaddyPageMode);
}
</script>

<template>
  <div class="mode-strip">
    <RadioGroup
      :value="props.mode"
      button-style="solid"
      @change="onModeChange"
    >
      <RadioButton v-for="item in props.modeOptions" :key="item.value" :value="item.value">
        {{ item.label }}
      </RadioButton>
    </RadioGroup>
    <span class="mode-summary">{{ summaries[props.mode] }}</span>
  </div>
</template>

<style scoped>
.mode-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
  padding: 10px 12px;
  border: 1px solid #eef0f4;
  border-radius: 8px;
  background: #fbfcfd;
}

.mode-summary {
  color: #667085;
  font-size: 12px;
}
</style>
