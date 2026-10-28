<script lang="ts" setup>
/**
 * 站点防护模式卡片选择弹窗（对齐雷池模式切换）。
 *
 * 三张卡对应站点自己的状态，各写清「选了之后会发生什么」：
 * 防护模式挂 waf_protect、观察模式挂 waf_protect_detect、维护模式写 503 响应。
 * 全部是站点块内的定点替换，需 diff 确认后保存，互不影响其他站点。
 *
 * 只负责展示与选择，不做请求：保存与确认由调用方掌握。
 */
import { computed } from 'vue';

import { Modal } from 'antdv-next';

import { protectModeOption } from '../protect-mode';
import {
  PROTECT_MODE_OPTIONS,
  type SiteProtectMode,
} from '../protect-mode';

const props = defineProps<{
  /** 当前生效的模式；用于高亮与「未改动不提交」判断 */
  current: SiteProtectMode;
  open: boolean;
  /** 提交中：禁用确认按钮并显示 loading */
  saving?: boolean;
}>();

const emit = defineEmits<{
  confirm: [mode: SiteProtectMode];
  'update:open': [value: boolean];
}>();

/** 待确认的模式；打开时由父组件传入的 current 派生 */
const selected = defineModel<SiteProtectMode>('selected', { required: true });

const changed = computed(() => selected.value !== props.current);

const currentLabel = computed(() => protectModeOption(props.current).label);

function choose(mode: SiteProtectMode) {
  if (props.saving) return;
  selected.value = mode;
}

function handleOk() {
  if (props.saving || !changed.value) return;
  emit('confirm', selected.value);
}
</script>

<template>
  <Modal
    :open="props.open"
    :width="760"
    title="防护模式"
    :ok-text="props.saving ? '应用中' : '继续'"
    cancel-text="取消"
    :confirm-loading="props.saving"
    :ok-button-props="{ disabled: !changed }"
    @update:open="emit('update:open', $event)"
    @ok="handleOk"
  >
    <p class="mode-scope">
      这三项只作用于当前站点，保存时会在站点块内定点改写，其他站点不受影响。
    </p>
    <p class="mode-current">
      当前：<span class="mode-current-value">{{ currentLabel }}</span>
    </p>

    <div class="mode-grid">
      <button
        v-for="option in PROTECT_MODE_OPTIONS"
        :key="option.value"
        type="button"
        class="mode-card"
        :class="[option.tone, { active: selected === option.value }]"
        :disabled="props.saving"
        @click="choose(option.value)"
      >
        <span class="mode-card-title">{{ option.label }}</span>
        <span class="mode-card-desc">{{ option.description }}</span>
      </button>
    </div>
  </Modal>
</template>

<style scoped>
.mode-scope {
  margin: 0 0 6px;
  color: hsl(var(--muted-foreground));
  font-size: 13px;
}

.mode-current {
  margin: 0 0 16px;
  color: hsl(var(--muted-foreground));
  font-size: 13px;
}

.mode-current-value {
  font-weight: 600;
  color: hsl(var(--foreground));
}

.mode-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

/* 卡片即「选项」：整块可点，避免小圆点难点中 */
.mode-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
  padding: 14px 16px 16px;
  font: inherit;
  text-align: left;
  cursor: pointer;
  background: hsl(var(--muted) / 30%);
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease,
    box-shadow 0.16s ease;
}

.mode-card:hover:not(:disabled) {
  border-color: hsl(var(--foreground) / 25%);
}

.mode-card:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.mode-card-title {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.3;
}

.mode-card-desc {
  font-size: 12.5px;
  line-height: 1.55;
  color: hsl(var(--muted-foreground));
}

/* 选中态用语义色描边 + 淡底；未选中的卡片保持中性，避免三色互抢 */
.mode-card.success.active {
  background: hsl(var(--success) / 10%);
  border-color: hsl(var(--success) / 60%);
  box-shadow: 0 0 0 2px hsl(var(--success) / 14%);
}

.mode-card.success.active .mode-card-title {
  color: hsl(var(--success));
}

.mode-card.warning.active {
  background: hsl(var(--warning) / 12%);
  border-color: hsl(var(--warning) / 62%);
  box-shadow: 0 0 0 2px hsl(var(--warning) / 16%);
}

.mode-card.warning.active .mode-card-title {
  color: hsl(var(--warning));
}

.mode-card.danger.active {
  background: hsl(var(--destructive) / 10%);
  border-color: hsl(var(--destructive) / 60%);
  box-shadow: 0 0 0 2px hsl(var(--destructive) / 14%);
}

.mode-card.danger.active .mode-card-title {
  color: hsl(var(--destructive));
}

@media (max-width: 720px) {
  .mode-grid {
    grid-template-columns: 1fr;
  }
}
</style>
