<script setup lang="ts">
import { computed } from 'vue';
import { Icon } from '@iconify/vue';
import type { StatCard } from '../data';

interface Props {
  data: StatCard;
}

const props = defineProps<Props>();

const safeColor = computed(() => props.data.color ?? '#6b7280');

const cardBg = computed(() => {
  const c = safeColor.value;
  return `linear-gradient(135deg, ${c}0d 0%, ${c}03 100%)`;
});

const iconWrapStyle = computed(() => {
  const c = safeColor.value;
  return {
    color: c,
    background: `linear-gradient(135deg, ${c}20, ${c}08)`,
    boxShadow: `0 0 20px ${c}18`,
  };
});

const trendDir = computed(() => props.data.trend?.dir);
const trendColor = computed(() => (trendDir.value === 'up' ? '#10b981' : '#ef4444'));
</script>

<template>
  <div
    class="stat-card group relative h-full overflow-hidden rounded-2xl p-5 transition-all duration-300 hover:shadow-lg hover:-translate-y-0.5"
    :style="{ background: cardBg }"
  >
    <div class="flex items-start justify-between">
      <div class="flex flex-col gap-1.5">
        <span class="text-sm text-gray-500 font-medium">{{ data.title }}</span>
        <div class="flex items-baseline gap-1.5">
          <span class="text-3xl font-bold" :style="{ color: safeColor }">{{ data.value }}</span>
          <span v-if="data.unit" class="text-xs text-gray-400">{{ data.unit }}</span>
        </div>
        <div v-if="data.trend" class="flex items-center gap-1 text-xs">
          <Icon :icon="trendDir === 'up' ? 'mdi:arrow-up' : 'mdi:arrow-down'" :style="{ color: trendColor }" />
          <span :style="{ color: trendColor }">{{ data.trend.value }}%</span>
        </div>
      </div>
      <div
        class="h-12 w-12 flex items-center justify-center rounded-full text-xl transition-transform duration-300 group-hover:scale-110"
        :style="iconWrapStyle"
      >
        <Icon :icon="data.icon || 'mdi:help-circle-outline'" />
      </div>
    </div>
    <div
      class="absolute h-28 w-28 rounded-full opacity-[0.04] transition-opacity duration-300 -bottom-6 -right-6 group-hover:opacity-[0.08]"
      :style="{ backgroundColor: safeColor }"
    />
  </div>
</template>

<style scoped>
.stat-card {
  border: 1px solid rgba(0, 0, 0, 0.04);
}
</style>
