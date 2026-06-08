<script setup lang="ts">
import { computed } from 'vue';
import { Icon } from '@iconify/vue';
import { useUserStore } from '@vben/stores';
import { Avatar } from 'ant-design-vue';

import { resolveAvatar } from '#/utils/avatar';

const userStore = useUserStore();

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
  stats: () => [],
});

const defaultColor = '#6b7280';
const defaultIcon = 'mdi:help-circle-outline';

const username = computed(() => userStore.userInfo?.username ?? '');
const userAvatar = computed(() =>
  resolveAvatar(userStore.userInfo?.avatar, userStore.userInfo?.username),
);
const displayName = computed(() => userStore.userInfo?.realName || username.value);
</script>

<template>
  <div class="banner-card overflow-hidden rounded-2xl p-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div class="flex items-center gap-4">
        <Avatar :size="48" :src="userAvatar" class="shrink-0">
          {{ displayName.slice(0, 1) }}
        </Avatar>
        <div>
          <h3 class="text-lg font-semibold">欢迎回来，{{ username }}</h3>
          <p class="text-sm text-gray-400 leading-7">统计范围：{{ props.rangeText }}</p>
        </div>
      </div>
      <div class="flex items-center gap-6">
        <div v-for="item in props.stats" :key="item.id" class="flex flex-col items-center gap-1">
          <div
            class="h-9 w-9 flex items-center justify-center rounded-full text-base"
            :style="{
              color: item.color || defaultColor,
              background: `linear-gradient(135deg, ${item.color || defaultColor}20, ${item.color || defaultColor}08)`,
              boxShadow: `0 0 16px ${item.color || defaultColor}15`,
            }"
          >
            <Icon :icon="item.icon || defaultIcon" />
          </div>
          <span class="text-lg font-bold" :style="{ color: item.color || defaultColor }">
            {{ item.value }}
          </span>
          <span class="text-xs text-gray-400">{{ item.label }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.banner-card {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.03) 0%, rgba(139, 92, 246, 0.02) 100%);
  border: 1px solid rgba(0, 0, 0, 0.04);
}
</style>
