<script setup lang="ts">
import { computed } from 'vue';
import { Icon } from '@iconify/vue';
import { useAppStore } from '@/store/modules/app';
import { useAuthStore } from '@/store/modules/auth';

const appStore = useAppStore();
const authStore = useAuthStore();

const gap = computed(() => (appStore.isMobile ? 0 : 16));

interface StatisticData {
  id: number;
  label: string;
  value: string | number;
  color?: string;
  icon?: string;
}

interface Props {
  rangeText: string;
  stats: StatisticData[];
}

const props = withDefaults(defineProps<Props>(), {
  rangeText: '',
  stats: () => []
});

const defaultColor = '#6b7280';
const defaultIcon = 'mdi:help-circle-outline';
</script>

<template>
  <NCard :bordered="false" class="banner-card overflow-hidden rounded-2xl">
    <NGrid :x-gap="gap" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem span="24 s:24 m:18">
        <div class="flex-y-center">
          <div
            class="h-72px w-72px flex-center shrink-0 rounded-full"
            style="background: linear-gradient(135deg, rgba(59, 130, 246, 0.12) 0%, rgba(139, 92, 246, 0.08) 100%)"
          >
            <Icon icon="mdi:account-circle-outline" class="text-40px text-primary" />
          </div>
          <div class="pl-12px">
            <h3 class="text-18px font-semibold">欢迎回来，{{ authStore.userInfo.username }}</h3>
            <p class="text-sm text-gray-400 leading-30px">统计范围：{{ props.rangeText }}</p>
          </div>
        </div>
      </NGridItem>
      <NGridItem span="24 s:24 m:6">
        <div class="flex items-center justify-end gap-6">
          <div v-for="item in props.stats" :key="item.id" class="flex flex-col items-center gap-1">
            <div
              class="h-9 w-9 flex items-center justify-center rounded-full text-base"
              :style="{
                color: item.color || defaultColor,
                background: `linear-gradient(135deg, ${item.color || defaultColor}20, ${item.color || defaultColor}08)`,
                boxShadow: `0 0 16px ${item.color || defaultColor}15`
              }"
            >
              <Icon :icon="item.icon || defaultIcon" />
            </div>
            <span class="font-number text-lg font-bold" :style="{ color: item.color || defaultColor }">
              {{ item.value }}
            </span>
            <span class="text-xs text-gray-400">{{ item.label }}</span>
          </div>
        </div>
      </NGridItem>
    </NGrid>
  </NCard>
</template>

<style scoped>
.banner-card {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.03) 0%, rgba(139, 92, 246, 0.02) 100%);
  border: 1px solid rgba(0, 0, 0, 0.04);
}
</style>
