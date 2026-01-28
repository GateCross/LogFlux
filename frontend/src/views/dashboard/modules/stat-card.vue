<script setup lang="ts">
import { computed } from 'vue';
import type { StatCard } from '../data';

interface Props {
  data: StatCard;
}

const props = defineProps<Props>();

const iconStyle = computed(() => {
  return {
    color: props.data.color,
    backgroundColor: `${props.data.color}1a`
  };
});
</script>

<template>
  <NCard class="h-full rounded-2xl border-none shadow-sm hover:shadow-md transition-shadow">
    <div class="flex items-center justify-between">
      <div class="flex flex-col gap-2">
        <span class="text-gray-500 text-sm font-medium">{{ data.title }}</span>
        <div class="flex items-baseline gap-1">
          <span class="text-2xl font-bold font-number">{{ data.value }}</span>
          <span v-if="data.unit" class="text-xs text-gray-400">{{ data.unit }}</span>
        </div>
      </div>
      <div 
        v-if="data.icon" 
        class="flex items-center justify-center w-12 h-12 rounded-xl text-xl"
        :style="iconStyle"
      >
        <div :class="data.icon || 'i-carbon:unknown'"></div>
      </div>
    </div>
  </NCard>
</template>

<style scoped>
.font-number {
  font-family: 'Outfit', sans-serif;
}
</style>
