<script setup lang="ts">
import { Icon } from '@iconify/vue/offline';

defineProps<{
  icon?: string;
  localIcon?: string;
  title: string;
}>();
</script>

<template>
  <div class="menu-title-cell">
    <Icon v-if="icon" :icon="icon" class="menu-icon" />
    <span class="menu-title-text">{{ title }}</span>
    <span v-if="localIcon && !icon" class="menu-icon-key">
      {{ localIcon }}
    </span>
  </div>
</template>

<style scoped>
.menu-title-cell {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
}

.menu-icon {
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
  color: #1677ff;
}

.menu-title-text {
  min-width: 0;
  overflow: hidden;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-icon-key {
  max-width: 88px;
  flex: 0 1 auto;
  overflow: hidden;
  padding: 1px 6px;
  color: #1677ff;
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: #e6f4ff;
  border: 1px solid #91caff;
  border-radius: 4px;
}
</style>
