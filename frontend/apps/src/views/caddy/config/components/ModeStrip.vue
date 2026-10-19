<script lang="ts" setup>
import type { CaddyPageMode } from '../types';

import { Button, RadioButton, RadioGroup } from 'antdv-next';

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
    <div class="mode-strip-left">
      <!-- 编辑视角：同一份 Caddyfile 的三种查看方式 -->
      <RadioGroup :value="props.mode" button-style="solid" @change="onModeChange">
        <RadioButton v-for="item in props.modeOptions" :key="item.value" :value="item.value">
          {{ item.label }}
        </RadioButton>
      </RadioGroup>
      <span class="mode-divider"></span>
      <!-- 防火墙是独立的业务功能，不与编辑视角并列成 Tab -->
      <Button
        :type="props.mode === 'waf' ? 'primary' : 'default'"
        @click="emit('change', 'waf')"
      >
        防火墙
      </Button>
    </div>
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
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  background: hsl(var(--muted) / 40%);
}

.mode-strip-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.mode-divider {
  width: 1px;
  height: 20px;
  background: hsl(var(--border));
}

.mode-summary {
  color: hsl(var(--muted-foreground));
  font-size: 12px;
}
</style>
